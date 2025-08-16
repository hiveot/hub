# tm2go

Generate go types, agent service handler and consumer handler from a WoT TM/TD document. Any security and communication data in the source is ignored as those are provided by the hub messaging layer. 

Usage: tm2go generate all package/tm/source.json [-o out] [-l lang]

```Arguments:
   generate  is the command to generate the go code and write it into the output directory
   all       generate the 'types', 'agent', and 'consumer' code
   tm.json   JSON file containing a Thing Model/Description document in the package tm directory.
```

```Options:
 -r, --recursive 
    Iterate directory tree in search of json files in tdd directories 
 -f, --force
    Force generating new API code even if it is newer
 --outdir "../api"
    Set the output directory to store the api code, relative to the TM/TD sourcefile.
 ```

This generates API source files in the output directory. The filenames consist of the
ThingID prefix and the suffix of: 'AgentAPI.go', 'ConsumerAPI.go', and 'Types.go' for
each type of API. 

### Examples

Generate the API of the Directory TD:
> tm2go go run cmd/tm2go/main.go generate all ./runtime/digitwin/tm/Directory.json 

Output:
> ./runtime/digitwin/api/AdminTypes.go
> ./runtime/digitwin/api/AdminAgentAPI.go
> ./runtime/digitwin/api/AdminConsumerAPI.go
```

```
Generate all API code and store them into the /tmp/tm2go directory:
> tm2go go run cmd/tm2go/main.go generate -outdir /tmp/tm2go -r all ./ 
 
```
