tcpkali --ws -c 100 -m '{"hdr":{"cmd":"echoRequest"},"payload":{"randomID": 1234}}' -r 10 127.0.0.1:80 -T 30
