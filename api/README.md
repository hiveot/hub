# API Directory

This directory contains API sources and generated code for golang, javascript and python.

The API generator can be found in the cmd directory under cmd/genapi.
This:
* Generates the WoT and HiveOT vocabulary from the yaml sources.
* Generates the Service APIs from their TDD documents

* cmd/genvocab generates vocabulary files from the yaml source
* cmd/genapi generates the client sources