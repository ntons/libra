db.apps.update(
    {
        "_id": "eff83ce8bd790069"
    },
    {
        "desc": "lhty2",
        "key": 10000001,
        "secret": "393424f62ceb82f2896a29598769db96",
        "fingerprint": "719c821c1cb785f73d7dd7229e7ea704",
        "permissions": [
            {
                "prefix": "/LHTY2."
            }
        ],
        "channels": {
            "dev": {
                "@type": "type.googleapis.com/sdk.Dev"
            },
            "dv": {
                "@type": "type.googleapis.com/sdk.Guest",
                "can_transfer": true
            },
            "gc": {
                "@type": "type.googleapis.com/sdk.Guest"
            },
            "gp": {
                "@type": "type.googleapis.com/sdk.Guest"
            }
        }
    },
    {
        "upsert": 1
    }
);
