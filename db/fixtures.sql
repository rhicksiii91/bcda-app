insert into acos(uuid, cms_id, name, client_id, public_key)
     values ('DBBD1CE1-AE24-435C-807D-ED45953077D3','A9995', 'ACO Lorem Ipsum', 'DBBD1CE1-AE24-435C-807D-ED45953077D3',
'-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
HwIDAQAB
-----END PUBLIC KEY-----'),
            ('0c527d2e-2e8a-4808-b11d-0fa06baf8254', 'A9994', 'ACO Dev', '0c527d2e-2e8a-4808-b11d-0fa06baf8254',
'-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
HwIDAQAB
-----END PUBLIC KEY-----');
-- The above public keys are paired with the published sample private key in shared_files/ATO_private.pem and are necessary
--   for unit, smoke, and Postman tests

insert into acos(uuid, cms_id, name, client_id, blacklisted)
     values ('A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', 'A8880', 'ACO Sit Amet', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', false),
            ('c14822fa-19ee-402c-9248-32af98419fe3', 'A8881', 'ACO Revoked',  'c14822fa-19ee-402c-9248-32af98419fe3', false),
            ('82f55b6a-728e-4c8b-807e-535caad7b139', 'T8882', 'ACO Not Revoked', '82f55b6a-728e-4c8b-807e-535caad7b139', false),
            ('3461C774-B48F-11E8-96F8-529269fb1459', 'A9990', 'ACO Small', '3461C774-B48F-11E8-96F8-529269fb1459', false),
            ('C74C008D-42F8-4ED9-BF88-CEE659C7F692', 'A9991', 'ACO Medium', 'C74C008D-42F8-4ED9-BF88-CEE659C7F692', false),
            ('8D80925A-027E-43DD-8AED-9A501CC4CD91', 'A9992', 'ACO Large', '8D80925A-027E-43DD-8AED-9A501CC4CD91', false),
            ('55954dba-d4d9-438d-bd03-453da4993fe9', 'A9993', 'ACO Extra Large', '55954dba-d4d9-438d-bd03-453da4993fe9', false),
            ('94b050bb-5a58-4f16-bd41-73a903977dfc', 'E9994', 'CEC ACO Dev', '94b050bb-5a58-4f16-bd41-73a903977dfc', false),
            ('749e6e2f-c45b-41d1-9226-8b7c54f96526', 'V994', 'NG ACO Dev', '749e6e2f-c45b-41d1-9226-8b7c54f96526', false),
            ('b8abdf3c-5965-4ae5-a661-f19a8129fda5', 'A9997', 'ACO Blacklisted', 'b8abdf3c-5965-4ae5-a661-f19a8129fda5', true);

