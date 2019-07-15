# Funk-agent

# Possible Enviorments to configure the Agent

TRACK_ALL =false | true (default ) => logs all finding Containers or only one with label funk.log=true
INSECURE_SKIP_VERIFY =  false (default)| true => disable ssl verifikation for server connection
FUNK_SERVER => Complete Server URL like (wss://localhost:3000)


# Possible Labels you can give the tracking dockercontainer

funk.log = false | true (default) => big lever log this container or not ? will only be affekt if TRACK_ALL is false
funk.log.stats = false | true (default)  => Log Stats info for this Container ?
funk.log.logs = false | true (default ) => Log Stdout for this Container ? 

