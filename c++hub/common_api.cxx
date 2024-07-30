#include <sstream>
#include <string>
#include <vector>
#include <time.h>
#include <map>

bool matchStrInList(const std::vector<std::string>& uriList,
                    const std::string& desiredStr)
{
    bool ret = false;
    for (const std::string& uri : uriList)
    {
        size_t pos = uri.find_last_of(':');
        std::string uuid = (pos != std::string::npos && pos + 1 < uri.length()) ? uri.substr(pos + 1) : "";
        if (uuid == desiredStr)
        {
            ret = true;
        }
    }
    return ret;
}

//void tokenizeString(const std::string& source, const char* delim, std::vector<std::string>& tokens,const size_t startPos = 0);
void tokenizeString(const std::string& source,
		    const char* delim,
		    std::vector<std::string>& tokens,
		    const size_t startPos)
{
    size_t i = startPos;
    size_t pos = source.find(delim,i);
    while (pos != std::string::npos)
    {
        tokens.push_back(source.substr(i, pos - i));
        i = ++pos;
        pos = source.find(delim, pos);

        if (pos == std::string::npos)
        {
            tokens.push_back(source.substr(i, source.length()));
        }
    }
}

bool isPresentInList(const std::vector<std::string>& inpList,
                     const std::string& strToBeChecked)
{
    if (std::find(inpList.begin(), inpList.end(), strToBeChecked) != inpList.end())
    {
        return true;
    }
    return false;
}

bool isSmpFoundInMpsList(const std::string& smpValue, const std::string& mpsValueList)
{
    bool isMatched = false;
    vector<std::string> mpsTokens;

    csb::tokenizeString(mpsValueList, ",", mpsTokens);
    if ((mpsTokens.size() > 0) && (!smpValue.empty()))
    {
        isMatched = isPresentInList(mpsTokens, smpValue);
    }

    return isMatched;
}

