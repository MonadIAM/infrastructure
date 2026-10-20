UPDATE identity.interface_client
SET auth_address = 'https://${APPLICATIONS_IP}:4002/docs/#identity/tag/authn/POST/api/v1/authn/password',
    redirect_uris = ARRAY[
        'https://${APPLICATIONS_IP}:4001/docs/',
        'https://${APPLICATIONS_IP}:4002/docs/',
        'https://${APPLICATIONS_IP}:4003/docs/',
        'https://${APPLICATIONS_IP}:4004/docs/',
        'https://${APPLICATIONS_IP}:4005/docs/'
    ],
    updated_at = CURRENT_TIMESTAMP,
    version = version + 1
WHERE id = '00000003-0000-4000-8000-130000000002'
  AND (
      auth_address IS DISTINCT FROM 'https://${APPLICATIONS_IP}:4002/docs/#identity/tag/authn/POST/api/v1/authn/password'
      OR redirect_uris IS DISTINCT FROM ARRAY[
          'https://${APPLICATIONS_IP}:4001/docs/',
          'https://${APPLICATIONS_IP}:4002/docs/',
          'https://${APPLICATIONS_IP}:4003/docs/',
          'https://${APPLICATIONS_IP}:4004/docs/',
          'https://${APPLICATIONS_IP}:4005/docs/'
      ]
  );
