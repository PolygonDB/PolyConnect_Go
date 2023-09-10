import json
from websocket import create_connection


ws = create_connection("ws://localhost:25565/ws")

ws.send(json.dumps(
    {
        'password': 'ExamplePS', 
        'dbname': 'ExampleDB',
        'location' :'Example',
        'action' : 'retrieve'
    }
))
print(ws.recv())