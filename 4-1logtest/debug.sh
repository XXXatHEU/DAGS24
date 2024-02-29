#!/bin/bash
rm test -f
#rm *.db

go build -gcflags "-N -l" -o test *.go 
./test
