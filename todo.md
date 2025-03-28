##todo
--------------------------
- range faker (can input in array of values and it will pick one at random)
- db connector service
- build script
- test with multiple files
- performance testing
- unit testing
- rename to spoof

###support both db connector will be reusable
- ```
  "cache":{
    "db_host": "",
    "db_port": "",
    "db_username":"",
    "db_password":"",
    "statement": "//SELECT cusomterid FROM account.customer as c LEFT OUTER JOIN account.movies as m WHERE m.customerid = c.customerid"
  },
  "fields": [
    { "name": "customerid", "type": "uuid", "seed_type":"db", "schema":"account", "table": "customer" }, //SELECT cusomterid FROM account.customer LIMIT 500;
    { "name": "updated_at", "type": "timestamp", "format": "02-01-06 15:04:05" },
    { "name": "customer_email", "type": "email", "format": "email" },
    { "name": "phonenumber", "type": "phone", "format": "phone" },
    { "name": "movieid", "type": "uuid", "seed_type":"db", "schema":"account", "table": "movie" }, //SELECT customerid FROM account.customer LIMIT 500;
    { "name": "movietitle", "type": "timestamp", "format": "02-01-06 15:04:05" },
    { "name": "runtime", "type": "email", "format": "email" },
    { "name": "custmermovielink", "type": "email", "format": "email", "foreignkey": "customerid", "seed_type":"db", "schema":"account", "table": "customer" }
  ]
},
```
