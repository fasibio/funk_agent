# Funk-agent

This is the Dockeragent for [funk-server](https://github.com/fasibio/funk-server). 

If you don´t know what funk is please read [here](https://github.com/fasibio/funk-server) first. 


You need one on each Host|Worker|Minion

You can configure each Agent different on each installation and combine the information to one Elasticsearch Stack.

## Possible Enviorments to configure the Agent

INSECURE_SKIP_VERIFY =  false (default)| true => disable ssl verifikation for server connection

FUNK_SERVER => Complete Server URL like (wss://localhost:3000)

LOG_STATS => Log statsinfo values: all or  no

## Possible Labels you can give the tracking dockercontainer

funk.log = false | true (default) => big lever. log this container or not ?

funk.log.stats = false | true (default)  => Log Stats info for this Container ?

funk.log.logs = false | true (default ) => Log Stdout for this Container ? 

funk.searchindex => the eleaticsearch index to log. It will generate to index for log and for stats info.  if empty it will use default_(logs|stats)


funk.log.formatRegex => funk logs json out of the box if your logs have an other format than json (the complete line will be logs out of the box in message) and you want to seperate it, you can give the format by regex and declarate submatches. 




## explained formatRegex 
For example you Loglines looking like this: 
```[negroni] 2019-08-12T13:38:52Z | 200 |      1.519074ms | localhost:3001 | POST /graphql```

you can give this container the label funk.log.formatRegex with value: 

```\\[[a-z]*\\] *(?P<time>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z) *\\| *(?P<status>[0-9]{3}) *\\| *[\\t] *(?P<request_ms>\\d.\\d+)ms *\\| *(?P<domain>.*) *\\| *(?P<method>(GET|POST|PUT|DELETE)) *(?P<message>.*)```

Attention if you want to test it at an online regextool like [regexr](https://regexr.com/4j31a) you have to replace ```\\``` with ```\```. 
The Double Backslash you need at your docker-compose file

In Kibana you will see now an logentry with logs.time, logs.status, logs.request_ms, logs.domain ....

