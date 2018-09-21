# schelly-postgres



`pg_dump` parameters that can be set:

```
General options:
  --file=FILENAME          output file or directory name

Options controlling the output content:
  --data-only              dump only the data, not the schema
  --encoding=ENCODING      dump the data in encoding ENCODING
  --schema-only            dump only the schema, no data  

Connection options:
  --dbname=DBNAME      database to dump (required)
  --host=HOSTNAME      database server host or socket directory (required)
  --port=PORT          database server port number
  --username=NAME      connect as specified database user (defaults to "postgres")
  --no-password        never prompt for password

Schelly postgres repo custom options:
  --password           password to be placed on ~/.pgpass (required)
```

`pg_dump` parameters that currently can't be set and the values that are used:

```
General options:
  --format=c|d|t|p         output file format (custom, directory, tar,
                               plain text (default))                               -> value used: tar
  --jobs=NUM               use this many parallel jobs to dump                 -> value used: 1
  --verbose                verbose mode                                        -> value used: --verbose  
  --compress=0-9           compression level for compressed formats            -> value used: 9
  --column-inserts             dump data as INSERT commands with column names      -> value used: --column-inserts
  --inserts                    dump data as INSERT commands, rather than COPY      -> value used: --inserts  
  --quote-all-identifiers      quote all identifiers, even if not key words        -> value used: --quote-all-identifiers

Options controlling the output content:
  --clean                  clean (drop) database objects before recreating     -> value used: --clean  
  --create                 include commands to create database in dump         -> value used: --create  
```

`pg_dump` parameters that currently can't be set but will soon be:
```
  --schema=SCHEMA          dump the named schema(s) only  
  --exclude-schema=SCHEMA  do NOT dump the named schema(s)  
  --table=TABLE            dump the named table(s) only  
  --exclude-table=TABLE    do NOT dump the named table(s)  
  --exclude-table-data=TABLE   do NOT dump data for the named table(s)  
```

# Known limitations

Currently this Repository supports only synchronous backup process






--------
This exposes the common functions of Backy2 with Schelly REST APIs so that it can be used as a backup backend for Schelly (backup scheduler), or standalone with curl, if you wish.

Backy2 is a great tool for creating Ceph image backups. For more, visit http://backy2.com/docs/

See more at http://github.com/flaviostutz/schelly

# Usage

docker-compose .yml

```
version: '3.5'

services:

  restic-api:
    image: flaviostutz/schelly-backy2
    ports:
      - 7070:7070
    environment:
      - LOG_LEVEL=debug
```

```
#create a new backup
curl -X POST localhost:7070/backups

#list existing backups
curl -X localhost:7070/backups

#get info about an specific backup
curl localhost:7070/backups/abc123

#remove existing backup
curl -X DELETE localhost:7070/backups/abc123

```

# REST Endpoints

As in https://github.com/flaviostutz/schelly#webhook-spec
