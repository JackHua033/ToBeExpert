#!/usr/bin/env python

import os
import re
import sys
import os.path

fcontent = ""
with open("imsi_msisdn.h", "r") as f:
    data = f.readlines()
    for index in range(len(data)):
        if ( index == 0 ):
            nline = data[index].strip()+"\n"+"#define IMSI_FILLIN 12345\n#define MSISDN_FILLIN 123456\n"
        elif ("MY_IMSI" in data[index]):
            nline = data[index].replace("FEAT_NUM", "IMSI_FILLIN")
        elif ("MY_MSISDN" in data[index]):
            nline = data[index].replace("FEAT_NUM", "MSISDN_FILLIN")
        else:
            nline = data[index]
        fcontent += nline 

with open("imsi_msisdn.h", "w") as nf:
    nf.write(fcontent)
