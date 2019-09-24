# Funk-agent

This is the Containeragent for [funk-server](https://github.com/fasibio/funk-server). 

If you don´t know what funk is please read [here](https://github.com/fasibio/funk-server) first. 

[![pipeline status](https://gitlab.com/fasibio/funk_agent/badges/master/pipeline.svg)](https://gitlab.com/fasibio/funk_agent/commits/master) [![coverage report](https://gitlab.com/fasibio/funk_agent/badges/master/coverage.svg)](https://sonar.server2.fasibio.de/dashboard?id=fasibio_funk_agent_master)



You need one on each Host|Worker|Minion

You can configure each Agent different on each installation and combine the information to one Elasticsearch Stack.

## Possible Environments to configure the Agent
 Envoirmentname | value | description | require
 ---            | ---   | ---         | ---  
FUNK_SERVER | wss://[url]:[port] | Complete Funk Server URL | true
CONNECTION_KEY | string | The Key to authenticate against [funk-server](https://github.com/fasibio/funk-server). Is declared at your funk-server | true
INSECURE_SKIP_VERIFY | false (default) or true | disable ssl verification for server connection | false
LOG_STATS | all cumulated(default) or no | this agent should be collect statsinformation (cumulated send the mostly needed Statsinfos like : RamUsageMb, CPUUsagePercent...) | false
SWARM_MODE | false (default) or true | Agent run on a swarm Cluster. Get better Metainformation about the Containers. | false
LOG_LEVEL | debug | info (default) | warn | error |Which log-level for the agent own logs | false
ENABLE_GEO_IP_INJECT  | false (default) or true | Will download a [geolite2](https://www.maxmind.com) DB to get geoinfomation by IP Adresses | false


## Possible Labels you can give each to tracking dockercontainer (by labels/annotation)

labelname | value | description
---  | --- | --- 
funk.log | boolean  (default true)  | big lever. log this container or not ?
funk.log.stats | boolean (default true)  | Log Stats info for this Container ?
funk.log.logs | boolean (default true) | Log Stdout/Stderr for this Container ? 
funk.log.staticcontent | json string | static information who whants to send for this container for example: {\"stage\": \"dev\"} (take a look for escaping inside docker-compose.yml or manifest.yml)
funk.searchindex | string | the eleaticsearch index to log. It will generate a index for log and for stats info.  if empty it will use default_(logs|stats)
funk.log.geodatafromip |string (starts with .)| is the path inside your log to the ipaddress where geodata will be inject. something like this ```.RequestAddr``` (at the moment only work with flat data on root level). You have to enable environment(**ENABLE_GEO_IP_INJECT**) at your funk_agent to use this flag.
funk.log.formatRegex | regex with subgroups | funk logs json out of the box. If your logs have a format other than json (the complete line will be logged to field message) and you want to separate it, you can give the format by regex and decelerate submatches. 


## example formatRegex 
For example you Loglines looking like this: 
```
  [negroni] 2019-08-12T13:38:52Z | 200 |      1.519074ms | localhost:3001 | POST /graphql
```

you can give this container the label funk.log.formatRegex with value: 

```
\\[[a-z]*\\] *(?P<time>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z) *\\| *(?P<status>[0-9]{3}) *\\| *[\\t]{0,} *(?P<request_time>\\d{0,5}.\\d+)(?P<request_format>(ms|µs)) *\\| *(?P<domain>.*) *\\| *(?P<method>(GET|POST|PUT|DELETE)) *(?P<message>.*)
```

**Attention** if you want to test it at an online regextool like [regexr](https://regexr.com/4j31a) you have to replace ```\\``` with ```\```. 
The Double Backslash you need at your docker-compose file

In Kibana you will see now an logentry with separate information: logs.time, logs.status, logs.request_ms, logs.domain ....


I am sure your regex would be better than this example. 

If you have build some Regex for standard logs like Apache, NGNIX, etc. I am happy to get Issue/Merge Request to add this to this Page. 

## Special at docker Swarm
Run it as mode *global*
At the container you have to set Container labels not deploy labels. (the labels at root)


## Dependencies

If you enable the flag enableGeoIPInject then this product includes GeoLite2 data created by MaxMind, available from
[https://www.maxmind.com](https://www.maxmind.com).
