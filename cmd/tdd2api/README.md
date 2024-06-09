# tdd2api

Generate an api from a WoT TD document.

Usage: tdd2api tdd.json [-o out] [-l lang]

```Arguments:
   tdd.json   JSON file containing a Thing Description document.
```

```Options:
 -o out
    output directory to write the generated api into. 
    Default is "./api/{lang}"
 -l lang
    language to generate. One of go, js, py3
    Default is all languages.
 ```

```Result:
This generates a source file in the output directory containing the name of the ThingID with the extension corresponding to the language.

For example:

> tdd2api runtime/authn/tdd/admin.json -l go

Output:
> ./api/go/authnAdmin.go

```
