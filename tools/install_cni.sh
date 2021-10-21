#! /bin/sh

mkdir -p /opt/cni/bin
mv /opt/cni/bin/wooshcni /opt/cni/bin/wooshcni-bak
# cp /usr/local/bin/wooshcni /opt/cni/bin/
zstd -d -o /opt/cni/bin/wooshcni /wooshcni.zst
mkdir -p /etc/cni/net.d

cat > /etc/cni/net.d/02-wooshnet.conflist <<EOF 
{
    "name":"wooshcni",
    "cniVersion":"0.4.0",
    "plugins":[
        {
            "cniVersion":"0.4.0",
            "type":"wooshcni",
            "name": "wooshcni"
        }
    ]
}
EOF
