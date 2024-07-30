#!/usr/bin/env bash
input_dir="$1"
if [ "$input_dir" == "" ]; then
	echo "ERROR: please input the dir name you want to search."
	exit 0
fi

# Find all TSV files under input directory
tsv_files=$(find "$input_dir" -type f -name "*.tsv")

# Loop over each TSV file
for tsv_file in $tsv_files; do
	manualrun_found=0
	# Check if the file has any lines with "Force Tags",
	# if it has MANUAL or MANUALRUN or DEVELOPMENT
	force_tags_line=$(grep -E "Force\ Tags" $tsv_file)
	if [[ "$force_tags_line" == *MANUAL* || "$force_tags_line" == *DEVELOPMENT* ]]; then
		manualrun_found=1
	fi

	# Check if the file has any lines with "sleep"
	grep -rnvE 'sleepTime|#|CommonData|Run Keyword|Documentation|AND|Console|sleeping|\.\.\.' "$tsv_file" | grep 'sleep' | while read line; do 
		line_num=$(echo $line |sed 's/ //g' | cut -d ':' -f1)
		sleep_value=$(echo $line |sed 's/ //g' | cut -d ':' -f2 | sed 's/sleep//g' | sed 's/\r//g')
		# Convert all value to int second to do compare
		if [[ "$sleep_value" =~ ^[0-9]+(\.[0-9]+)?s$ ]]; then
			sleep_value=$(echo "$sleep_value" | sed 's/s$//')
		elif [[ "$sleep_value" =~ ^[0-9]+(\.[0-9]+)?ms$ ]]; then
			sleep_value=$(echo "$sleep_value" | sed 's/ms$//' | awk '{printf "%d", $1/1000}')
		fi

		# Check if the sleep value is greater than 5 seconds
		if (( $(echo "$sleep_value > 5" | bc -l) )); then
			# If "Force Tags" is not found or it doesn't have MANUAL or MANUALRUN or DEVELOPMENT,
			# check the [Tags] in same FC123_XXXX if it has MANUAL or MANUALRUN or DEVELOPMENT
			if (( manualrun_found == 0 )); then
				tags_line=$(sed -n "$((line_num-line_num+1)),$line_num p" $tsv_file | grep '\[Tags\]' | tail -n 1)
				if [[ "$tags_line" != *MANUAL* && "$tags_line" != *DEVELOPMENT* ]]; then
					suitename=$(echo ${tsv_file##*/} | cut -d '.' -f1)
					findsuite=$(grep $suitename suite_owners.txt)
					if [ "$findsuite" != "" ]; then
					    owner=${findsuite##*:}
					fi
					echo "$tsv_file:$line:$owner"
				fi
			fi
		fi
	done
done
