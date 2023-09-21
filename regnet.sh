#!/bin/bash

# Check if at least two parameters are provided
if [[ $# -lt 2 ]]; then
    echo "Usage:"
    echo "$0 start <option>"
    echo "OR"
    echo "$0 cli <params> <command> [command_flags]"
    echo ""
    echo "start options are: node1, node2, node3"
    echo "cli params are: node1, node2, node3"
    exit 1
fi

# Based on the first parameter
if [[ "$1" == "start" ]]; then
    case "$2" in
        "node1")
            ilxd --regtest --regtestval --loglevel=debug
            ;;
        "node2")
            ilxd --regtest --loglevel=debug --listenaddr=/ip4/127.0.0.1/tcp/9004 --grpclisten=/ip4/127.0.0.1/tcp/5002 --datadir="$HOME/.ilxd2"
            ;;
        "node3")
            ilxd --regtest --loglevel=debug --listenaddr=/ip4/127.0.0.1/tcp/9005 --grpclisten=/ip4/127.0.0.1/tcp/5003 --datadir="$HOME/.ilxd3"
            ;;
        *)
            echo "Invalid option for start. Use 'node1', 'node2', or 'node3'."
            exit 1
            ;;
    esac

elif [[ "$1" == "cli" ]]; then
    if [[ $# -lt 3 ]]; then
        echo "Usage for cli: $0 cli <params> <command> [command_flags]"
        exit 1
    fi

    case "$2" in
        "node1")
            shift 2
            ilxcli --serveraddr=/ip4/127.0.0.1/tcp/5001 --rpccert="$HOME/.ilxd/rpc.cert" "$@"
            ;;
        "node2")
            shift 2
            ilxcli --serveraddr=/ip4/127.0.0.1/tcp/5002 --rpccert="$HOME/.ilxd2/rpc.cert" "$@"
            ;;
        "node3")
            shift 2
            ilxcli --serveraddr=/ip4/127.0.0.1/tcp/5003 --rpccert="$HOME/.ilxd3/rpc.cert" "$@"
            ;;
        *)
            echo "Invalid params for cli. Use 'node1', 'node2', or 'node3'."
            exit 1
            ;;
    esac

else
    echo "Invalid command. Use 'start' or 'cli'."
    exit 1
fi
