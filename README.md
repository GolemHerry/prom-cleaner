# prom-cleaner
> an efficient tool designed to clean prometheus TSDB


## Usage

### **A**. clean all metrics within a period of time
> Just set `cleaner.from` and `cleaner.to`


### **B**. clean specified metrics
> Based on **A**, just set `cleaner.metrics`


### **C**, clean metrics those containing the specified labels
> Based on **A**, just set `cleaner.labels`


### **D**, clean specified metrics containing the specified labels
> The above combination, both set `cleaner.metrics` and `cleaner.labels`
