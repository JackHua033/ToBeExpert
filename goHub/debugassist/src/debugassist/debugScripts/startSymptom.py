#!/usr/bin/env python3
'''
# Description - trigger symptom log collection
#
'''

## IMPORTS ##
import os
import sys
import argparse
import time
import subprocess
import re
from datetime import datetime

def usageMsg():
    msg = "\n"\
          "   1. Trigger application CNF services to execute default debug commands. \n"\
          "      /usr/bin/python3 /opt/SMAW/INTP/bin64/startSymptom.py\n"\
          "   2. Trigger application CNF specific services(with -s filter) or specific pods(with -p filter) to execute specific debug commands(with -l filter) \n"\
          "      /usr/bin/python3 /opt/SMAW/INTP/bin64/startSymptom.py -s <SERVICE NAME1> <SERVICE NAME2> -p <POD NAME1> <POD NAME2> -l <LABEL NAME1> <LABEL NAME2>\n"\
          "   3. Trigger application CNF specific services to execute specific debug commands with json format file \n"\
          "      /usr/bin/python3 /opt/SMAW/INTP/bin64/startSymptom.py -f **.json\n"\
          "   4. Show all the debug commands supported by application CNF in json format. \n"\
          "      /usr/bin/python3 /opt/SMAW/INTP/bin64/startSymptom.py -o\n\n"\
          "Notes:\n"\
          "1.The -s/-l option cannot be delivered together with the -f option. If the -s/-l filter option and the -f input file option are delivered at the same time, the input file will be used and the -s/-l filter will be ignored.\n"\
          "2.The -o option cannot be delivered together with other options.\n"
    return msg

if "__main__" == __name__:
    parser = argparse.ArgumentParser(description = usageMsg(), formatter_class = argparse.RawDescriptionHelpFormatter)
    parser.add_argument("-o", "--output", action='store_true', default=False, help="Show all the debug commands supported by application CNF in json format.")
    parser.add_argument("-s", "--services", dest="services", nargs='+', help="Trigger debug commands with specific services")
    parser.add_argument("-p", "--pods", dest="pods", nargs='+', help="Trigger the debug commands with specific pods.")
    parser.add_argument("-l", "--labels", dest="labels", nargs='+', help="Trigger the debug commands with labels filter.")
    parser.add_argument("-f", "--file", dest="file", type=str, help="Trigger debug commands with json format file.")
    opt = parser.parse_args()
    #1. Decode parameters
    queryList = ""
    if opt.services:
        queryList = "service=" + ",".join(str(i) for i in opt.services)
    podsList = ""
    if opt.pods:
        podsList = ",".join(str(i) for i in opt.pods)
        if queryList:
            queryList = queryList + "\&podid=" + podsList
        else :
            queryList = "podid=" + podsList
    labelsList = ""
    if opt.labels:
        labelsList = ",".join(str(i) for i in opt.labels)
        if queryList:
            queryList = queryList + "\&label=" + labelsList
        else :
            queryList = "label=" + labelsList
        
    filePostion = ""
    if opt.file:
        if not os.path.exists(opt.file):
            print("File ", opt.file, " not exist.")
            sys.exit(-1)
        filePostion = opt.file

    #2. Get TLS info
    tlsFlag = os.getenv('DEBUG_ASSIST_HTTP_SERVER_TLS')
    tlsInfo = ""
    httpVersion = "http://"
    if tlsFlag == "true":
        keyFile = os.getenv('DEBUG_ASSIST_KEY_FILE')
        certFile = os.getenv('DEBUG_ASSIST_CERTS_FILE')
        cacertFile = os.getenv('DEBUG_ASSIST_CACERTS_FILE')
        certsPath = os.getenv('DEBUG_ASSIST_CERTS_MOUNT_PATH')
        tlsInfo = "--cacert " + certsPath + cacertFile + " --key " + certsPath + keyFile + " --cert " + certsPath + certFile
        httpVersion = "https://"

    # 3. Generate curl command
    podip = os.getenv('POD_IP')
    port = "8090"
    portEnv = os.getenv('DEBUG_ASSIST_SERVER_PORT')
    if portEnv != "":
        port = portEnv
    isIpv6Flag = os.getenv('IPV6_INT_IF_ENABLED')
    if isIpv6Flag == "true":
        httpAddr = httpVersion + "[" + podip + "]" + ":" + port
    else :
        httpAddr = httpVersion + podip + ":" + port
    if opt.output:
        curlCommand = "curl -v %s -X GET %s/api/ssd/v1/config"%(tlsInfo, httpAddr)
    elif filePostion:
        curlCommand = "curl -v %s -F -f=@%s -X POST %s/api/ssd/v1/config"%(tlsInfo, filePostion, httpAddr)
    elif queryList: 
        curlCommand = "curl -v %s -X POST %s/api/ssd/v1/config?%s"%(tlsInfo, httpAddr, queryList)
    else :
        curlCommand = "curl -v %s -X POST %s/api/ssd/v1/config"%(tlsInfo, httpAddr)
    print("Http request is: \n", curlCommand)
    print("Running command...")
    # 4. Get http response
    p = subprocess.Popen(curlCommand,stdout=subprocess.PIPE, stderr=subprocess.PIPE, stdin=subprocess.PIPE, shell=True)
    output, err = p.communicate()
    err = err.decode()
    for line in err.splitlines():
        if re.search('curl: \([1-9]\)', line):
            print("Curl command execute failed ,error code is: ", line)
        elif "< HTTP" in line:
            print("Http response is:", line.split('< ', 1)[1])
    output = output.decode()
    if output:
        print("Http response message is: ", output)
