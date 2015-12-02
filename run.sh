docker run -t -i --rm -p 8000:8000 -p 80:80 \
    -e MARATHON_ENDPOINT='http://marathon01.dal.moz.com:8080,http://marathon02.dal.moz.com:8080,http://marathon03.dal.moz.com:8080' \
    -e BAMBOO_ENDPOINT=http://192.168.48.142:8000 \
    -e BAMBOO_ZK_HOST='zk01.dal.moz.com,zk02.dal.moz.com,zk03.dal.moz.com' \
    -e BAMBOO_ZK_PATH=/bamboo \
    -e CONFIG_PATH='/var/bamboo/config/production.example.json' \
    docker-repo-host.dal.moz.com:5000/bamboo-v2 bash
    #bamboo -bind=":8000" \
