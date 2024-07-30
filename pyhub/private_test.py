#!/usr/bin/env python

import os
import re
import sys
import os.path

def check_be_type():
    udmplutofile = 'pluto.h'
    udmplutofile = '/home/testrunner/5G_CDCommon/include/testbed/tcl/UDM/pluto.h'
    if not os.path.exists(udmplutofile):
        print "ERROR: " + udmplutofile + " is not exist!"
        sys.exit(1)

    #check the BE_TYPE
    betype = ''
    fp = open(udmplutofile, 'r+')
    linecontext = fp.readlines()
    for index in range(len(linecontext)):
        if (linecontext[index].find(";") < 0 and re.search("BE_TYPE", linecontext[index])):
            betype = linecontext[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')[1].replace('"', '')
    return betype

def get_config_file_and_tenant_name():
    onendspath = ''
    tenantname = ''
    filename = '/home/testrunner/5G_CDCommon/include/user_conf.h'
    fp = open(filename, 'r+')
    lines = fp.readlines()
    for index in range(len(lines)):
        if (re.search("One-NDS configuration given with full pat", lines[index]) and lines[index+1].find(";") < 0):
            onendspath = lines[index+1].replace(' ', '').replace('\t', '').replace('\r', '').strip()
        elif (lines[index].find(";") < 0 and re.search("TENANT", lines[index])):
            tenantname = lines[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')[1].replace('"', '')
    return onendspath, tenantname

def get_parameters_via_betype(betype, configfile):
    pgdip = ""
    pgduser = ""
    pgdpw = ""
    sdlvnfid = ""
    fp = open(configfile, 'r+')
    linecontext = fp.readlines()
    for index in range(len(linecontext)):
        if (linecontext[index].find(";") < 0 and re.search("ONENDS_PGD_IP", linecontext[index])):
            pgdip = linecontext[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')[1].replace('"', '')
        elif (linecontext[index].find(";") < 0 and re.search("ONENDS_PGD_LDAP_USER", linecontext[index])):
            pgduser = linecontext[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')[1].replace('"', '')
        elif (linecontext[index].find(";") < 0 and re.search("ONENDS_PGD_LDAP_PW", linecontext[index])):
            pgdpw = linecontext[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')[1].replace('"', '')
        elif (betype =='SDL' and linecontext[index].find(";") < 0 and re.search("SDL_VNF_ID", linecontext[index])):
            sdlvnfid = linecontext[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')[1].replace('"', '')
    return pgdip, pgduser, pgdpw, sdlvnfid


def get_onends_parameters():
    configfile, tenant = get_config_file_and_tenant_name()
    if configfile == '':
        print "Error: not find One-NDS config file."
        sys.exit(1)

    pgdip, pgduser, pgdpw, sdlvnfid = get_parameters_via_betype('OneNDS', configfile)
    sncmd, msincmd = pack_cmd_via_bitype(pgdip, pgduser, pgdpw, tenant, 'OneNDS', sdlvnfid)
    print sncmd
    print msincmd
    return true


def get_sdl_parameters():
    configfile, tenant = get_config_file_and_tenant_name()
    if configfile == '':
        print "Error: not find SDL config file."
        sys.exit(1)

    pgdip, pgduser, pgdpw, sdlvnfid = get_parameters_via_betype('SDL', configfile)
    return true

def pack_cmd_via_bitype(pgdip, pgduser, pgdpw, tenant, bitype, sdlvnfid):
    sncmd = ""
    msincmd = ""
    commonsncmd = "intDataName=snLngt, dataType=integerType, nodeName=POD, nodeName="+tenant+", nodeName=HLR_SUBSCRIBER, dc=APPLICATIONS, dc=CONFIGURATION, "
    commonmsincmd = "intDataName=msinLength, dataType=integerType, nodeName=POD, nodeName="+tenant+", nodeName=SUBSCRIBER, dc=APPLICATIONS, dc=CONFIGURATION, "
    if bitype == 'SDL':
        if sdlvnfid == '':
            print "Error: not configure SDL_VNF_ID for SDL."
            sys.exit(1)
        sncmd = 'ldapsearch -x -h '+pgdip+' -p 16611 -D cn='+pgduser+' -w '+pgdpw+' -b '+ '\"'+commonsncmd+ 'vnfId=' +sdlvnfid+ ', dc=PGW, dc=C-NTDB'+'\" | grep intDataValue | awk \'{print $2}\''
        msincmd = 'ldapsearch -x -h '+pgdip+' -p 16611 -D cn='+pgduser+' -w '+pgdpw+' -b '+ '\"'+commonmsincmd+ 'vnfId=' +sdlvnfid+ ', dc=PGW, dc=C-NTDB'+'\" | grep intDataValue | awk \'{print $2}\''
    else:
        sncmd = 'ldapsearch -x -h '+pgdip+' -p 16611 -D cn='+pgduser+' -w '+pgdpw+' -b '+ '\"'+commonsncmd+'dc=PGW, dc=C-NTDB'+'\" | grep intDataValue | awk \'{print $2}\''
        msincmd = 'ldapsearch -x -h '+pgdip+' -p 16611 -D cn='+pgduser+' -w '+pgdpw+' -b '+ '\"'+commonmsincmd+'dc=PGW, dc=C-NTDB'+'\" | grep intDataValue | awk \'{print $2}\''
    return sncmd, msincmd

def get_feature_num_length():
    msinLength = ""
    snLngt = ""
    pgdip = ""
    pgduser = ""
    pgdpw = ""

    #get parameters via OneNDS or SDL
    betypename = check_be_type()
    if betypename == 'OneNDS':
        get_onends_parameters()
    elif betypename == 'SDL':
        get_sdl_parameters()
    else:
        print "ERROR: invalid BE_TYPE= " + betypename
        sys.exit(1) 

    sys.exit(0)

    filename = get_config_file_and_tenant_name()
    if filename == '':
        print "Error: not find one-nds config full path"
        sys.exit(1)

    fp = open(filename, 'r+')
    linecontext = fp.readlines()
    for index in range(len(linecontext)):
        contextlist = linecontext[index].replace(' ', '').replace('\t', '').replace('\r', '').strip().split('=')
        if contextlist[0] == "ONENDS_PGD_IP":
            pgdip = contextlist[1].replace('"', '')
        elif contextlist[0] == "ONENDS_PGD_LDAP_USER":
            pgduser = contextlist[1].replace('"', '')
        elif contextlist[0] == "ONENDS_PGD_LDAP_PW":
            pgdpw = contextlist[1].replace('"', '')

    sncmd = 'ldapsearch -x -h '+pgdip+' -p 16611 -D cn='+pgduser+' -w '+pgdpw+' -b "intDataName=snLngt, dataType=integerType, nodeName=POD, nodeName=DEFAULT, nodeName=HLR_SUBSCRIBER, dc=APPLICATIONS, dc=CONFIGURATION, dc=PGW, dc=C-NTDB"|grep intDataValue|awk \'{print $2}\''
    if (os.popen(sncmd).read().strip() == ''):
        print "Error: can not get the snLength"
        sys.exit(1)
    snLngt = os.popen(sncmd).read().strip()

    msincmd = 'ldapsearch -x -h '+pgdip+' -p 16611 -D cn='+pgduser+' -w '+pgdpw+' -b "intDataName=msinLength, dataType=integerType, nodeName=POD, nodeName=DEFAULT, nodeName=SUBSCRIBER, dc=APPLICATIONS, dc=CONFIGURATION, dc=PGW, dc=C-NTDB"|grep intDataValue|awk \'{print $2}\''
    if (os.popen(msincmd).read().strip() == ''):
        print "Error: can not get the msinLength"
        sys.exit(1)
    msinLength = os.popen(msincmd).read().strip()
    return msinLength, snLngt

def update_imsi_msisdn_file(imsi_str, msisdn_str, imsi_msisdn_file):
    newcontent = ""
    with open(imsi_msisdn_file, "r") as f:
        origcontent = f.readlines()
        for index in range(len(origcontent)):
            if ( index == 0 ):
                nline = origcontent[index].strip() + "\n" + imsi_str + msisdn_str
            elif ("#define IMSI_FILLIN" in origcontent[index]):
                continue
            elif ("#define MSISDN_FILLIN" in origcontent[index]):
                continue
            elif ("MY_IMSI" in origcontent[index]):
                nline = origcontent[index].replace("FEAT_NUM", "IMSI_FILLIN")
            elif ("MY_MSISDN" in origcontent[index]):
                nline = origcontent[index].replace("FEAT_NUM", "MSISDN_FILLIN")
            else:
                nline = origcontent[index]
            newcontent += nline

    with open(imsi_msisdn_file, "w") as nf:
        nf.write(newcontent)

def update_file_with_search_result(input_file_list):
    for index in range(len(input_file_list)):
        feat_num = ""
        cmd = 'grep FEAT_NUM ' + input_file_list[index] + ' | grep -v COMMDB'
        feat_line = os.popen(cmd).read().strip()
        if feat_line == '':
            cmd1 = 'grep FEATURE_NUMBER ' + cur_path + '/CommonData/feature_variables.h | grep -v FEAT_NUM'
            new_feat_line = os.popen(cmd1).read().strip()
            if new_feat_line == '':
                print "Error: No FEATURE_NUMBER exists!"
                sys.exit(1)
            feat_num = new_feat_line.split(' ')[2]
        else:
            feat_num = feat_line.split(' ')[2]

        msin_real_len = int(msin_len)- 4
        snlngt_real_len = int(snlngt_len)- 4
        mgap = len(feat_num)- msin_real_len
        sgap = len(feat_num)- snlngt_real_len
        if mgap > 0:
           msin_str = '#define IMSI_FILLIN ' + feat_num[:-mgap] + '\n'
        else:
           msin_str = '#define IMSI_FILLIN ' + feat_num + '\n'

        if sgap > 0:
           snlngt_str = '#define MSISDN_FILLIN ' + feat_num[:-sgap] + '\n'
        else:
           snlngt_str = '#define MSISDN_FILLIN ' + feat_num + '\n'

        update_imsi_msisdn_file(msin_str, snlngt_str, input_file_list[index])

def main():
    global cur_path
    global msin_len
    global snlngt_len
    #get current path
    cur_path = os.getcwd()

    #get length of msinLength, snLngt
    msin_len, snlngt_len = get_feature_num_length()

    #find imsi_msisdn.h
    cmd = 'find ' + cur_path + ' -name imsi_msisdn.h'
    filelist = os.popen(cmd).read().strip().split('\n')
    if filelist == '':
        print "Error: No imsi_msisdn.h file exists under: " + cur_path
        sys.exit(1)

    #update each searched file
    update_file_with_search_result(filelist)

if __name__=='__main__':
    #main(sys.argv)
    main()


