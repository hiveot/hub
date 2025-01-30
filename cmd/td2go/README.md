# td2go

Generate go types, agent service handler and consumer handler from a WoT TD document.

Usage: td2go generate all package/tdd/source.json [-o out] [-l lang]

```Arguments:
   generate  is the command to generate the go code and write it into the output directory
   all       generate the 'types', 'agent', and 'consumer' code
   tdd.json  JSON file containing a Thing Description document in the package tdd directory.
```

```Options:
 -r, --recursive 
    Iterate directory tree in search of json files in tdd directories 
 -f, --force
    Force generating new API code even if it is newer
 --outdir "../api"
    Set the output directory to store the api code, relative to the TD sourcefile.
 ```

This generates API source files in the output directory. The filenames consist of the
ThingID prefix and the suffix of: 'AgentAPI.go', 'ConsumerAPI.go', and 'Types.go' for
each type of API. 

### Examples

Generate the API of the Directory TD:
> td2go go run cmd/td2go/main.go generate all ./runtime/digitwin/tdd/Directory.json 

Output:
> ./runtime/digitwin/api/AdminTypes.go
> ./runtime/digitwin/api/AdminAgentAPI.go
> ./runtime/digitwin/api/AdminConsumerAPI.go
```

```
Generate all API code and store them into the /tmp/td2go directory:
> td2go go run cmd/td2go/main.go generate -outdir /tmp/td2go -r all ./ 
 
```
