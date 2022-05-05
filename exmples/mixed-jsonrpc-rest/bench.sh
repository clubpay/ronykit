#tcpkali --ws --dump-one -c 1000 -m '{"hdr": {"cmd": "echoRequest"}, "payload": {"randomID": 1234}}' -r 100 127.0.0.1:7080
#tcpkali --ws -c 1000 -m '{"hdr": {"cmd": "echoRequest"}, "payload": {"randomID": 1234}}' -r 100 127.0.0.1:7080 -T 30
#tcpkali --ws -c 100 -m '{"hdr":{"cmd":"echoRequest"},"payload":{"randomID": 1234}}' -r 1k 127.0.0.1:80 -T 30
tcpkali --ws -c 100 -m '{"hdr":{"cmd":"echoRequest"},"payload":{"randomID": 1234}}' -r 1k 127.0.0.1:81/ws -T 30
