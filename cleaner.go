package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	SIZE                      = 10
	GET_ALL_METRICS_URL       = "/api/v1/label/__name__/values"
	GET_METRICS_BY_LABELS_URL = "/api/v1/label/%s/%s"
	SWEEP_URL                 = "/api/v1/admin/tsdb/clean_tombstones"
	DELETE_URL                = "/api/v1/admin/tsdb/delete_series"
)

type Prometheus struct {
	Host string `yaml:"host"`
	TLS  interface{}
}

type Cleaner struct {
	Prometheus `yaml:"prometheus"`
	From       time.Time         `yaml:"from"`
	To         time.Time         `yaml:"to"`
	Metrics    []string          `yaml:"metrics"`
	Labels     map[string]string `yaml:"labels"`
	Timeout    int               `yaml:"timeout"`
}

type Job struct {
	Cleaner `yaml:"cleaner"`
}

func newCleaner(jobs string) (Cleaner, error) {
	data, err := ioutil.ReadFile(jobs)
	if err != nil {
		log.Errorf("failed to open job file, error:%v\n", err)
		return Cleaner{}, err
	}
	var job Job
	if err := yaml.Unmarshal(data, &job); err != nil {
		log.Errorf("failed to unmarshal job file, error:%v\n", err)
		return Cleaner{}, err
	}
	return job.Cleaner, nil
}

func (c *Cleaner) Do() error {
	if len(c.Metrics) == 0 && len(c.Labels) == 0 {
		c.Metrics = c.getAllMetrics()
	} else if len(c.Metrics) == 0 {
		c.Metrics = c.getMetricsByLabels()
	}
	query := c.parse()
	if query == "" {
		return errors.New("invalid queries\n")
	}
	if err := c.delete(query); err != nil {
		return err
	}
	return nil
}

func (c *Cleaner) parse() string {
	labelString := ""
	var labels []string
	for k, v := range c.Labels {
		labels = append(labels, k+"=\""+v+"\"")
	}
	if len(labels) != 0 {
		labelString = strings.Join(labels, ",")
		labelString = "{" + labelString + "}"
	}

	for i, v := range c.Metrics {
		c.Metrics[i] = "match[]=" + v + labelString
	}

	from := strconv.FormatInt(c.From.Unix(), 10)
	to := strconv.FormatInt(c.To.Unix(), 10)

	if c.From.Unix() > 0 && c.To.Unix() > 0 {
		return "&start=" + from + "&end=" + to
	}
	return ""
}

func (c *Cleaner) delete(query string) error {
	for len(c.Metrics) > SIZE {
		if err := c.do(DELETE_URL, strings.Join(c.Metrics[:SIZE], "&")+query); err != nil {
			return err
		}
		if err := c.sweep(); err != nil {
			return err
		}

		c.Metrics = c.Metrics[SIZE:]
	}
	return c.do(DELETE_URL, strings.Join(c.Metrics, "&")+query)
}

func (c *Cleaner) sweep() error {
	return c.do(SWEEP_URL, "")
}

func (c *Cleaner) do(url, query string) error {
	client := &http.Client{}

	if query != "" {
		url += "?" + query
	}
	log.Println("http://" + c.Host + url)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*time.Duration(c.Timeout))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+c.Host+url, nil)
	if err != nil {
		log.Errorf("bad job, error: %v\n", err)
		return err
	}

	res := &http.Response{}
	go func() {
		res, err = client.Do(req)
		if err != nil {
			log.Errorf("bad job: clean, error: %v", err)
		}
		cancel()
	}()

	<-ctx.Done()
	if ctx.Err() != context.Canceled {
		log.Errorf("%#v\n", ctx.Err())
		return ctx.Err()
	}
	if res.StatusCode == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("failed to delete, status code:%d", res.StatusCode)
}

func (c *Cleaner) getAllMetrics() []string {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://"+c.Host+GET_ALL_METRICS_URL, nil)
	if err != nil {
		log.Errorf("bad job, error: %v\n", err)
		return nil
	}
	res, err := client.Do(req)
	if err != nil {
		log.Errorf("bad job: get all metrics, error: %v", err)
	}
	var metrics struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&metrics); err != nil {
		log.Errorf("%v\n", err)
	}
	return metrics.Data
}

func (c *Cleaner) getMetricsByLabels() []string {
	client := &http.Client{}

	metricsSlice := make([]string, 0)
	for k, v := range c.Labels {
		req, err := http.NewRequest(http.MethodGet, "http://"+c.Host+fmt.Sprintf(GET_METRICS_BY_LABELS_URL, k, v), nil)
		if err != nil {
			log.Errorf("bad job, error: %v\n", err)
			return nil
		}
		res, err := client.Do(req)
		if err != nil {
			log.Errorf("bad job: get all metrics, error: %v", err)
		}
		var metrics struct {
			Status string   `json:"status"`
			Data   []string `json:"data"`
		}
		if err := json.NewDecoder(res.Body).Decode(&metrics); err != nil {
			log.Errorf("%v\n", err)
		}
		metricsSlice = append(metricsSlice, metrics.Data...)
	}
	return metricsSlice
}
