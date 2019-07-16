# Funk-agent

## Possible Enviorments to configure the Agent

INSECURE_SKIP_VERIFY =  false (default)| true => disable ssl verifikation for server connection

FUNK_SERVER => Complete Server URL like (wss://localhost:3000)

LOG_STATS => Log statsinfo values: all or  no

## Possible Labels you can give the tracking dockercontainer

funk.log = false | true (default) => big lever. log this container or not ?

funk.log.stats = false | true (default)  => Log Stats info for this Container ?

funk.log.logs = false | true (default ) => Log Stdout for this Container ? 

funk.searchindex => the eleaticsearch index to log. It will generate to index for log and for stats info.  if empty it will use default_(logs|stats)