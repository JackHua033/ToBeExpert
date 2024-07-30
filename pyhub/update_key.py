
#!/usr/bin/env python

import os
import re
import sys
import os.path

def update_robot_file(robot_file, line_num):
    newcontent = ""
    with open(robot_file, "r") as f:
        origcontent = f.readlines()
        for index in range(len(origcontent)):
            nline = origcontent[index]
            if (str(index+1) == line_num):
                if ("sorAckIndication" in origcontent[index]):
                    nline = nline.replace("sorAckIndication", "sorackindication")
                if ("accessTechList" in origcontent[index]):
                    nline = nline.replace("accessTechList", "accesstechlist")
                if ("sorSendingTime" in origcontent[index]):
                    nline = nline.replace("sorSendingTime", "sorsendingtime")
                if ("ackInd" in origcontent[index]):
                    nline = nline.replace("ackInd", "ackind")
                if ("provisioningTime" in origcontent[index]):
                    nline = nline.replace("provisioningTime", "provisioningtime")
                if ("sorMacIausf" in origcontent[index]):
                    nline = nline.replace("sorMacIausf", "sormaciausf")
            newcontent += nline

    with open(robot_file, "w") as nf:
        nf.write(newcontent)

def update_file_with_search_result(input_file_list):
    for index in range(len(input_file_list)):
        cmd = 'grep -rn Nsoraf_get_sor_information_response ' + input_file_list[index] + '| grep -v html | grep -v xml'
        feat_line = os.popen(cmd).read().strip()
        feat_num = feat_line.split(':')[0]
        update_robot_file(input_file_list[index], feat_num)

    for index2 in range(len(input_file_list)):
        cmd2 = 'grep -rn Nudm_SDM_am_data_update_sor_response ' + input_file_list[index2] + '| grep -v html | grep -v xml'
        feat_line2 = os.popen(cmd2).read().strip()
        feat_num2 = feat_line2.split(':')[0]
        update_robot_file(input_file_list[index2], feat_num2)

def main():
    global cur_path
    cur_path = os.getcwd()

    #find robot file
    cmd = 'find ' + cur_path + ' -name FC123_109724_UDM_FT_TC*.robot'
    filelist = os.popen(cmd).read().strip().split('\n')
    if filelist == '':
        print "Error: No FC123_109724_UDM_FT_TCxxx.robot file exists under: " + cur_path
        sys.exit(1)

    #update each searched file
    update_file_with_search_result(filelist)

if __name__=='__main__':
    main()
