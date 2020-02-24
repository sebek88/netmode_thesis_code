#!/bin/bash

DATE="$(date +"%Y-%m-%d %T.%6N")"

i=0
j=0
bridge_counter=0
OVSbridges="$(sudo ovs-vsctl list-br)"
bridge_counter="$(sudo ovs-vsctl list-br | wc -l)"

#OVSDB="$(sudo ovs-ofctl dump-flows s1 | grep -v reply)"

IFS=' ' read -ra bridge_array <<< $OVSbridges

<< --Comment-- 
while [ $bridge_counter -ge 1 ] 
do 
    ((bridge_counter--))
    echo "${bridge_array[$((j))]}"
    OVSDB="$(sudo ovs-ofctl dump-flows ${bridge_array[$((j))]} | grep -v reply)"  
    ((j++))
done

if [ -n "$OVSbridges" ]
then
    bridge="$(sudo ovs-vsctl list-br | head -n 1)"
fi
--Comment--


while [ $bridge_counter -ge 1 ]
do
    ((bridge_counter--))
#    echo "${bridge_array[$((j))]}"
    OVSDB="$(sudo ovs-ofctl dump-flows ${bridge_array[$((j))]} | grep -v reply)" 
        
    if [ -n "$OVSDB" ]
    then {
        while read -r line
        do
            ((i++))
            cookie="$(echo "$line" | awk -F 'cookie=' '{print $2 "="}' | cut -f 1 -d ",")"
            duration="$(echo "$line" | awk -F 'duration=' '{print $2 "="}' | cut -f 1 -d "s")"
            table="$(echo "$line" | awk -F 'table=' '{print $2 "="}' | cut -f 1 -d ",")"
            n_packets="$(echo "$line" | awk -F 'n_packets=' '{print $2 "="}' | cut -f 1 -d ",")"
            n_bytes="$(echo "$line" | awk -F 'n_bytes=' '{print $2 "="}' | cut -f 1 -d ",")"
            idle_age="$(echo "$line" | awk -F 'idle_age=' '{print $2 "="}' | cut -f 1 -d ",")"
            actions="$(echo "$line" | awk -F 'actions=' '{print $2 " "}' )"

            echo -e "\nThis is now the flow: $i in bridge: ${bridge_array[$((j))]}\n"

            echo "$DATE: flow: flow$i | cookie: $cookie | duration: $duration | table: $table | n_packets: $n_packets | n_bytes: $n_bytes | idle_age: $idle_age | actions: $actions"

            echo "The chaincode is going to run now to add flow$i"

            docker exec cli peer chaincode invoke -n mychainc -C mychannel -c '{"Args":["initFlow","'"flow$i"'","'"$actions"'","42","lala","'"$cookie"'","'"$duration"'","'"$table"'","'"$n_packets"'","'"$n_bytes"'","'"$idle_age"'"]}'

            echo "the chaincode finished now with invocation of initFlow"

        done <<<"$OVSDB"
    }

    else
        echo "OVSDB is empty and there are no flows in the Database"
    fi
    ((j++))
done


