# Examples of querying for TD documents


## JSONPath

W3C's WoT specifies that things can be queried using JSONPath. This documents provides some examples. See the Directory Service test cases for a working example.  



1. Query things by ID (this is kinda useless as the API provides this, but it is a nice simple example)
> $[?(@.id=="thing1")]

2. Query by Thing type 'sensor'
> $[?(@['@type']=='sensor')]



References

1. JsonPath online evaluator: http://jsonpath.com/

