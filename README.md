# 此為備份mongodb與es的程式

參數如下：

cron job 哪個時間點做備份，預設凌晨3點

```
--schedule
```

本機或是container上備份的父目錄，預設 /data/backup，程式預設有遞迴建立目錄，不用額外建立

```
--local_parent_dir
```

保留database份數，預設是3天，最新的將取代最舊的

```
--retention
```

sftp server放database備份的目錄，除了在本機或是container上，備份預設的3天之外，sftp server也會備份1份，
程式預設有建立目錄，不用額外建立

```
--remote_parent_dir
```

sftp server位址，請使用 IP/hostname:port格式，預設localhost:22

```
--host
```

sftp server的使用者帳號

```
--user
```

sftp server的使用者密碼

```
--password
```

是否開啟mongodb備份，預設true

```
--mongo_enable
```

mongo server位址，因為部署在k8s上，所以我設計的service name預設是mongodb，這邊也是如此

```
--mongo_addr
```

mongo server port，預設27017

```
--mongo_port
```

是否開啟es db備份，預設true

```
--es_enable
```

es client位址，因為部署在k8s上，所以我設計的es client service name預設是es-client，這邊也是如此

```
--es_addr
```

es client port，預設9200

```
--es_port
```

# 手動 mongo 備份與還原

**這個備份方式會把所有mongo db都備份下來**

```
mongodump --archive=/path/mongodb_backup.gz --gzip --host mongodb --port 27017
```

**如果要指定db name**

```
mongodump --db <yourdb> --archive=/path/mongodb_backup.gz --gzip --host mongodb --port 27017
```

**還原方式，前者全部還原，後者指定db name**

```
gunzip -c /path/mongodb_backup.gz | mongorestore --archive
gunzip -c /path/mongodb_backup.gz | mongorestore --archive -d <yourdb>
```

# 手動 es 備份與還原

**修改es config**

使用alpine image，檔案在/usr/share/elasticsearch/config/elasticsearch.yml，master/data server都要！

```
path.repo: ["/snapshot/backups/my_backup"]
```

**啟動es之前要做的事情**

在pod寫的方式可以用如下，加在deployment裡面，這是master:

```
          command:
            - sh
            - -c
          args:
            - 'chown elasticsearch:elasticsearch {{ .Values.volumes.snap.mountPath }} &&
               echo path.repo: ["{{ .Values.volumes.snap.mountPath }}"] >> /usr/share/elasticsearch/config/elasticsearch.yml &&
              /docker-entrypoint.sh elasticsearch
              {{ printf "-Dcluster.name=%s" .Values.config.cluster.name }}
              -Dnode.master=true
              -Dnode.data=false
              -Dhttp.enabled=false
              -Ddiscovery.zen.ping.unicast.hosts={{ .Values.service.master.name }}'

```

這是data statefulset:

```
          command:
            - sh
            - -c
          args:
            - 'echo path.repo: ["{{ .Values.volumes.snap.mountPath }}"] >> /usr/share/elasticsearch/config/elasticsearch.yml &&
              /docker-entrypoint.sh elasticsearch
              {{ printf "-Dcluster.name=%s" .Values.config.cluster.name }}
              -Dnode.master=false
              -Dnode.data=true
              -Dhttp.enabled=false
              -Ddiscovery.zen.ping.unicast.hosts={{ .Values.service.master.name }}'
```

**註冊snapshot repository**

記得於container裡要先建立/snapshot/backups/my_backup，且owner權限是elasticsearch

```
PUT /_snapshot/my_backup
{
    "type": "fs",
    "settings": {
        "location": "/snapshot/backups/my_backup",
        "compress": true
    }
}
```

**建立名為snapshot_1的snapshot於my_backup dir**

程式裡我選擇了用wait的方式，總不能還沒跑完備份，就進行archive與壓縮。備份完的es snapshot會落在es data container上，而不是在這個程式所在的位子，
也因此helm在寫的時候，才需要pvc來分享備份的資料

```
PUT /_snapshot/my_backup/snapshot_1?wait_for_completion=true
PUT /_snapshot/my_backup/snapshot_1
```

**還原es，於另外一個cluster**

請先把上敘備份的es snapshot(/snapshot/backups/my_backup) copy到另外一台es server，
路徑也使用一樣比較好，/snapshot/backups/my_backup，owner權限是elasticsearch

**修改es config**

參考上面步驟

**啟動es之前要做的事情**

參考上面步驟

**註冊snapshot repository**

參考上面步驟

**restore snapshot**

```
POST /_snapshot/my_backup/snapshot_1/_restore
```

參考：
* https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html
