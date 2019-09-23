# Funk-agent

This is the Containeragent for [funk-server](https://github.com/fasibio/funk-server). 

If you don´t know what funk is please read [here](https://github.com/fasibio/funk-server) first. 

[![pipeline status](https://gitlab.com/fasibio/funk_agent/badges/master/pipeline.svg)](https://gitlab.com/fasibio/funk_agent/commits/master) [![coverage report](https://gitlab.com/fasibio/funk_agent/badges/master/coverage.svg)](https://sonar.server2.fasibio.de/dashboard?id=fasibio_funk_agent_master)



You need one on each Host|Worker|Minion

You can configure each Agent different on each installation and combine the information to one Elasticsearch Stack.

## Possible Environments to configure the Agent

INSECURE_SKIP_VERIFY =  false (default)| true => disable ssl verifikation for server connection

FUNK_SERVER => Complete Server URL like (wss://localhost:3000)

CONNECTION_KEY ==> The Key to authenticate against [funk-server](https://github.com/fasibio/funk-server). 

LOG_STATS => Log statsinfo values: all or  no

SWARM_MODE => false (default) | true => Agent run on a swarm Cluster. Get better Metainformation about the Containers.

LOG_LEVEL => debug | info (default) | warn | error => Which log-level for the agent own logs


## Possible Labels you can give the tracking dockercontainer

funk.log = false | true (default) => big lever. log this container or not ?

funk.log.stats = false | true (default)  => Log Stats info for this Container ?

funk.log.logs = false | true (default ) => Log Stdout/Stderr for this Container ? 

funk.log.staticcontent = json string => static information who whants to send for this container

funk.searchindex => the eleaticsearch index to log. It will generate a index for log and for stats info.  if empty it will use default_(logs|stats)

funk.log.geodatafromip => is the path at log to the ipaddress where geodata will be inject => something like this .RequestAddr (at the moment only work with flat data on root level)

funk.log.formatRegex => funk logs json out of the box. If your logs have a format other than json (the complete line will be logged to field message) and you want to separate it, you can give the format by regex and decelerate submatches. 




## example formatRegex 
For example you Loglines looking like this: 
```[negroni] 2019-08-12T13:38:52Z | 200 |      1.519074ms | localhost:3001 | POST /graphql```

you can give this container the label funk.log.formatRegex with value: 

```\\[[a-z]*\\] *(?P<time>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z) *\\| *(?P<status>[0-9]{3}) *\\| *[\\t]{0,} *(?P<request_time>\\d{0,5}.\\d+)(?P<request_format>(ms|µs)) *\\| *(?P<domain>.*) *\\| *(?P<method>(GET|POST|PUT|DELETE)) *(?P<message>.*)```

**Attention** if you want to test it at an online regextool like [regexr](https://regexr.com/4j31a) you have to replace ```\\``` with ```\```. 
The Double Backslash you need at your docker-compose file

In Kibana you will see now an logentry with separate information: logs.time, logs.status, logs.request_ms, logs.domain ....


I am sure your regex would be better than this example. 

If you have build some Regex for standard logs like Apache, NGNIX, etc. I am happy to get Issue/Merge Request to add this to this Page. 

## Dependencies

If you enable the flag enableGeoIPInject then this product includes GeoLite2 data created by MaxMind, available from
[https://www.maxmind.com](https://www.maxmind.com).