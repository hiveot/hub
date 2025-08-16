# API Directory

This directory contains API sources and generated code for golang, javascript and python.

The API generator can be found in the cmd/ directory.
This:
* Generates the WoT and HiveOT vocabulary from the yaml sources.
* Generates the Service APIs from their TM/TD documents

* cmd/genvocab generates vocabulary files from the yaml source
* cmd/tm2go generates go client and server sources from the TM/TD (experimental)