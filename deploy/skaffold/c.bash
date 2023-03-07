#!/bin/bash
 cat demo.txt |sort -nr | awk -F'[:.]' '$3$4!=p&&p=$3$4' >> abc.txt
