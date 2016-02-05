# Postback Delivery

Postback Delivery is a web-based app that ingests, queues, and delivers incoming http requests. All of the code has been developed and maintained by Mark Wunsch. 

# Table of Contents
-----
1. [Installation/Configuration](#installationconfiguration)
   * [Technology Stack](#technologystack)
   * [Installation](#installation)
   * [Configuration](#configuration)
1. [Usage](#usage)
   * [Data Flow](#dataflow)
   * [Sample Request](#samplerequest)
   * [Sample Response](#sampleresponse)
   * [Test Case](#testcase)
1. [Components](#components)
   * [Ingestion Agent (PHP)](#ingestionagent)
   * [Delivery Queue (Redis)](#deliveryqueue)
   * [Delivery Agent (GO)](#deliveryagent)
1. [Future Work](#futurework)
   * [Planned Features](#plannedfeatures)
   * [Wish List](#wishlist)

-----

# Installation/Configuration
-----

Everything you should need to get this web app installed and functional on a new system.

## Technology Stack

1. [Ubuntu](http://www.ubuntu.com/download)
1. [Apache2](https://httpd.apache.org/download.cgi#apache24)
1. [PHP](http://php.net/downloads.php)
1. [phpredis](https://github.com/phpredis/phpredis)
1. [Redis](http://redis.io/download)
1. [Redigo](https://golang.org/)
1. [GO](https://golang.org/)

## Installation

The following commands were used to install the necessary software on an already existing instance of Ubuntu:

Apache2
~~~
apt-get install apache2
~~~
PHP
~~~
apt-get install pgp5 libapache2-mod-php5
~~~
Redis
~~~
wget http://download.redis.io/redis-stable.tar.gz
tar xvzf redis-stable.tar.gz
cd redis-stable
make && make install
~~~
phpredis (after downloading source from github)
~~~
phpize
./configure
make && make install
~~~
GO (might only work on newer versions of Ubuntu)
~~~
apt-get install golang
~~~
Redigo
~~~
go get github.com/garyburd/redigo/redis
~~~

## Configuration

### Apache
Add 'ingest.php' to /etc/apache2/mods-enabled/dir.conf
Run "a2enmod rewrite" command to enable mod_rewrite
Add the following to .htaccess file in site's directory root:
~~~
<IfModule mod_rewrite.c>
RewriteEngine On
RewriteBase /
RewriteCond ${REQUEST_FILENAME} !-f
RewriteCond ${REQUEST_FILENAME} !-d
RewriteRule . /ingest.php [L]
<IfModule>
~~~
In /etc/apache2/apache2.conf, set "AllowOverride" to "all"

### phpredis
Add "extension=redis.io" to php.ini file

### GO
The following assumes that GO was installed to /usr/lib/go and that work will be done in $HOME/work.
Run the following commands:
~~~
export GOROOT=/usr/lib/go
export GOPATH=$HOME/work
export GOBIN=$GOPATH/bin
~~~

# Usage

## Data Flow
1. Web request >
2. [Ingestion Agent (php)](#ingestionagent) >
3. [Delivery Queue (redis)](#deliveryqueue) >
4. [Delivery Agent (GO)](#deliveryagent) >
5. Web response

## Components

### Ingestion Agent
The Ingestion Agent consists of the following components:
- ingest.php
   - This file accepts the incoming web request and intializes a new endpointRequest with the contents of the request.
- endpointRequest.php
   - This class is the main worker for the ingestion agent. It creates a new dataRequest for each individual data item and handles the interaction with Redis.
- dataRequest.php
   - This helper class keeps track of each individual data point from the original web request. One postback will be created for each dataRequest.

### Delivery Queue
The delivery queue is created and maintained in an instance of Redis. It contains the following data structures:
- Pending (List of UUIDs) 
   - Maintains a list of all UUIDs, each of which corresponds with a unique postback request.
   - Utilized as a First-In, First-Out (FIFO) list though the use of "LPUSH" for adding and "RPOP" for retrieving. 
- Values (Hash)
   - UUID from Pending list can be hashed to retrieve that URL of the postback request.
   - "UUID:method" can be hashed to retrieve the method of the postback request.
- Stats (Hash)
   - All time values stored in Stats are in milliseconds in Unix time.
   - "UUID:start" can be hashed to retrieve the time the postback request was received.
   - "UUID:post" can be hashed to retrieve the time the postback request was added to Pending list in Redis.
- Working (Sorted Set)
   - When the delivery agent begins processing a postback request, that UUID is added to this set with the time it began processing.
   - If a postback request is succesfully handled by the delivery agent, its UUID will be removed from Working
   - If the delivery agent fails during the processing of a request, its UUID will remain on this list to be retrieved and reprocessed.
- Delayed (Sorted Set) NOT YET IMPLEMENTED
   - This sorted set can be used to handle postback requests that are to be delivered after some configurable delay. 
   - Moving a UUID from this set to Pending would begin the processing of that request.

### Delivery Agent

