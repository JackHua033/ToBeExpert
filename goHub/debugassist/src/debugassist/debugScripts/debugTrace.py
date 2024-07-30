#!/usr/bin/env python3
'''
# Description - debug trace log collection
#
'''

## IMPORTS ##
import os
import sys
import argparse
import time
import subprocess
import signal
from datetime import datetime

def term_Handler(signal_num, frame): 
    file_obj.writelines("Get TERM signal: " + str(signal_num) + "\n")
    # Stop trace
    debugTraceStopCmd = "source ~/.bash_profile; CnAdminTool TraceStop " + sessionName
    p1 = subprocess.Popen(debugTraceStopCmd,stdout=subprocess.PIPE, stderr=subprocess.PIPE, stdin=subprocess.PIPE, shell=True)
    output1, _ = p1.communicate()
    output1 = output1.decode()
    file_obj.writelines("Debug trace stop command is: " + debugTraceStopCmd + "\n")
    file_obj.writelines("Debug trace stop log is: \n")
    file_obj.writelines(output1 + "\n")
    # Close log file
    file_obj.close()
    sys.exit(1)

def getInterval(interval): 
    maxInterval = 300
    if interval < 0:
        file_obj.writelines("Input interval is " + str(interval) + ", which is less than zero, currently not supported")
        file_obj.close()
        sys.exit(1)
    if interval > maxInterval :
        file_obj.writelines("Input interval is " + str(interval) + ", greater than the maximum allowed time interval, forcibly rewrite the interval to the maximum allowed time interval.\n")
        return maxInterval
    return interval

if "__main__" == __name__:
    parser = argparse.ArgumentParser()
    parser.add_argument("-i", "--interval", dest="interval", type=int, help="How long to collect debug logs. If interval is set, will ignore the start/stop input.")
    parser.add_argument("-start", "--start", action='store_true', default=False, help="The start flag for collecting debug trace.")
    parser.add_argument("-stop", "--stop", action='store_true', default=False, help="The stop flag for collecting debug trace.")
    parser.add_argument("-p", "--process", dest="process", nargs='+', help="Which process's log to collect. Must be set when interval or start is enabled.")
    opt = parser.parse_args()
    # Write log file
    currTime = datetime.now().strftime('_%Y%m%d_%H%M%S')
    file_path = "debugTraceExe.log" + currTime
    global file_obj
    file_obj = open(file_path, mode='w')

    currPath = os.getcwd()
    global sessionName
    sessionName = currPath.split("/").pop()

    if  opt.start and opt.stop:
        file_obj.writelines("The start flag and stop flag cannot be set at the same time. \n")
        file_obj.close()
        sys.exit(1)

    startFlag = False
    stopFlag = False
    waitIntervalFlag = False
    file_path = "/logstore/debugassist/debugTraceSession.log"
    if opt.interval:
        file_obj.writelines("Get input: interval is " + str(opt.interval) + " , monitor processes are " + str(opt.process) + "\n")
        if not opt.process:
            file_obj.writelines("ERROR! Must set the target processes to collect debug trace when interval is set. \n")
            file_obj.close()
            sys.exit(1)
        processList = " ".join(str(i) for i in opt.process)
        interval = getInterval(opt.interval)
        startFlag = True
        stopFlag = True
        waitIntervalFlag = True
    elif opt.start:
        file_obj.writelines("Get input: startFlag is " + str(opt.start) + " , monitor processes are " + str(opt.process) + "\n")
        if not opt.process:
            file_obj.writelines("ERROR! Must set the target processes to collect debug trace when start flag is true. \n")
            file_obj.close()
            sys.exit(1)
        # if debugTraceSession.log exist, Check whether there is currently a running session
        if os.path.exists(file_path):
            if os.path.getsize(file_path) != 0:
                file_obj.writelines("ERROR! There is already one running debug trace task, so cannot start another one. \n")
                file_obj.close()
                sys.exit(1)
        processList = " ".join(str(i) for i in opt.process)
        startFlag = True
        # save the sessionName, so stop session can get this sessionName
        with open(file_path, 'w') as file:
            file.write(sessionName)
    elif opt.stop:
        file_obj.writelines("Get input: stopFlag is " + str(opt.stop) + " , monitor processes are " + str(opt.process) + "\n")
        stopFlag = True
        # check if file existed
        if os.path.exists(file_path):
            # read session name
            with open(file_path, 'r') as file:
                sessionName = file.read()
            # clean the existed file
            with open(file_path, 'w') as file:
                file.write("")
        else:
            file_obj.writelines("There is no open debug trace session, so no need to stop debug trace. \n")
            file_obj.close()
            sys.exit(1)
    else:
        file_obj.writelines("There is no valid input, you must set interval or start/stop flag. \n")
        file_obj.close()
        sys.exit(1)

    file_obj.writelines("The session name is " + sessionName + "\n")

    for sig in [signal.SIGINT, signal.SIGHUP, signal.SIGTERM, signal.SIGTSTP]:
        signal.signal(sig, term_Handler)

    # Start trace
    if startFlag:
        debugTraceStartCmd = "source ~/.bash_profile; CnAdminTool TraceStart " + sessionName + " ALL_6 " + processList
        p = subprocess.Popen(debugTraceStartCmd,stdout=subprocess.PIPE, stderr=subprocess.PIPE, stdin=subprocess.PIPE, shell=True)
        output, err = p.communicate()
        output = output.decode()
        file_obj.writelines("Debug trace start command is: " + debugTraceStartCmd + "\n")
        file_obj.writelines("Debug trace start log is: \n")
        file_obj.writelines(output + "\n")

    # Wait log collection
    if waitIntervalFlag:
        file_obj.writelines("log collection wait " + str(interval) + " seconds" + "\n")
        time.sleep(interval)

    # Stop trace
    if stopFlag:
        debugTraceStopCmd = "source ~/.bash_profile; CnAdminTool TraceStop " + sessionName
        p1 = subprocess.Popen(debugTraceStopCmd,stdout=subprocess.PIPE, stderr=subprocess.PIPE, stdin=subprocess.PIPE, shell=True)
        output1, err = p1.communicate()
        output1 = output1.decode()
        file_obj.writelines("Debug trace stop command is: " + debugTraceStopCmd + "\n")
        file_obj.writelines("Debug trace stop log is: \n")
        file_obj.writelines(output1 + "\n")
        # change the log trace directory name from start session name to stop session name
        if opt.stop:
            try:
                oldDebugTraceSession = "/writeablelayer/oam/99/trace/" + sessionName + ".backup"
                newDebugTraceSession = "/writeablelayer/oam/99/trace/" + currPath.split("/").pop() + ".backup"
                os.rename(oldDebugTraceSession, newDebugTraceSession)
                print(f"Folder name changed from '{oldDebugTraceSession}' to '{newDebugTraceSession}'")
            except FileNotFoundError:
                print(f"Folder '{oldDebugTraceSession}' not found")

    # Close log file
    file_obj.close()
