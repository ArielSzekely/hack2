#!/bin/bash

IP_PATH=/home/arielck/workspace/projects/research/iproute2/iproute2-6.5.0/ip/ip

GUEST_NETNS_NAME=s1
VTEP_NETNS_NAME=s2

LINK_NAME_1=veth0-test
LINK_NAME_2=veth1-test

echo "----- Create VTEP netns"
$IP_PATH netns add $VTEP_NETNS_NAME
echo "----- Create guest netns"
time $IP_PATH netns add $GUEST_NETNS_NAME
echo "----- Create veth pair"
time $IP_PATH link add $LINK_NAME_1 type veth peer name $LINK_NAME_2
echo "----- Move veth peer to guest netns"
time $IP_PATH link set $LINK_NAME_1 netns $GUEST_NETNS_NAME
echo "----- Move veth peer to VTEP netns"
time $IP_PATH link set $LINK_NAME_2 netns $VTEP_NETNS_NAME
echo "----- Cleanup"
$IP_PATH netns delete $GUEST_NETNS_NAME
$IP_PATH netns delete $VTEP_NETNS_NAME
echo "----- Done"
