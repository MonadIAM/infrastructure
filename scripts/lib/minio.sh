require_minio_env() {
    require_env \
        KUBE_CONTEXT \
        MINIO_MC_IMAGE \
        MINIO_AUDIT_LOG_BUCKET \
        MINIO_CHANGE_LOG_BUCKET \
        MINIO_AUDIT_LOG_RETENTION_DAYS \
        MINIO_CHANGE_LOG_RETENTION_DAYS
}

wait_minio_bootstrap_pod_completed() {
    namespace="$1"
    name="$2"
    attempt=0

    while :; do
        phase="$(kubectl --context "$KUBE_CONTEXT" get pod "$name" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
        case "$phase" in
            Succeeded)
                kubectl --context "$KUBE_CONTEXT" logs "pod/$name" -n "$namespace"
                kubectl --context "$KUBE_CONTEXT" delete pod "$name" -n "$namespace" --ignore-not-found
                return
                ;;
            Failed)
                kubectl --context "$KUBE_CONTEXT" logs "pod/$name" -n "$namespace" || true
                kubectl --context "$KUBE_CONTEXT" describe pod "$name" -n "$namespace" || true
                exit 1
                ;;
        esac

        attempt=$((attempt + 1))
        if [ "$attempt" -ge 120 ]; then
            kubectl --context "$KUBE_CONTEXT" logs "pod/$name" -n "$namespace" || true
            kubectl --context "$KUBE_CONTEXT" describe pod "$name" -n "$namespace" || true
            exit 1
        fi
        sleep 5
    done
}

bootstrap_minio() {
    require_minio_env

    namespace=monadiam-data
    name=minio-main-bootstrap

    kubectl --context "$KUBE_CONTEXT" delete pod "$name" -n "$namespace" --ignore-not-found
    kubectl --context "$KUBE_CONTEXT" apply -f - <<-EOF
	apiVersion: v1
	kind: Pod
	metadata:
	  name: $name
	  namespace: $namespace
	  labels:
	    app.kubernetes.io/name: minio-main-bootstrap
	    app.kubernetes.io/component: object-storage-bootstrap
	    app.kubernetes.io/part-of: monadiam
	    app.kubernetes.io/managed-by: shell
	spec:
	  restartPolicy: Never
	  enableServiceLinks: false
	  containers:
	    - name: mc
	      image: $MINIO_MC_IMAGE
	      imagePullPolicy: IfNotPresent
	      env:
	        - name: HOME
	          value: /tmp
	        - name: MINIO_ENDPOINT
	          value: https://minio.monadiam-data.svc.cluster.local
	        - name: MINIO_AUDIT_LOG_BUCKET
	          value: $MINIO_AUDIT_LOG_BUCKET
	        - name: MINIO_CHANGE_LOG_BUCKET
	          value: $MINIO_CHANGE_LOG_BUCKET
	        - name: MINIO_AUDIT_LOG_RETENTION_DAYS
	          value: "$MINIO_AUDIT_LOG_RETENTION_DAYS"
	        - name: MINIO_CHANGE_LOG_RETENTION_DAYS
	          value: "$MINIO_CHANGE_LOG_RETENTION_DAYS"
	      command:
	        - /bin/sh
	        - -eu
	        - -c
	        - |
	          mkdir -p /tmp/.mc/certs/CAs
	          cp /mnt/minio-tls/ca.crt /tmp/.mc/certs/CAs/monadiam-root-ca.crt
	
	          . /mnt/minio-root/config.env
	
	          MINIO_ARCHIVER_ACCESS_KEY="\$(cat /mnt/minio-archiver/MINIO_ARCHIVER_ACCESS_KEY)"
	          MINIO_ARCHIVER_SECRET_KEY="\$(cat /mnt/minio-archiver/MINIO_ARCHIVER_SECRET_KEY)"
	          export MINIO_ARCHIVER_ACCESS_KEY MINIO_ARCHIVER_SECRET_KEY
	
	          i=0
	          until mc alias set local "\$MINIO_ENDPOINT" "\$MINIO_ROOT_USER" "\$MINIO_ROOT_PASSWORD" >/dev/null 2>&1; do
	              i=\$((i + 1))
	              if [ "\$i" -ge 60 ]; then
	                  echo "MinIO did not become ready in time: \$MINIO_ENDPOINT" >&2
	                  exit 1
	              fi
	              sleep 2
	          done
	
	          configure_archive_bucket() {
	              bucket="\$1"
	              days="\$2"
	
	              mc ls "local/\$bucket" >/dev/null 2>&1 || mc mb --with-lock "local/\$bucket"
	              mc retention set --default COMPLIANCE "\${days}d" "local/\$bucket"
	
	              {
	                  printf '%s\n' '{'
	                  printf '%s\n' '  "Rules": ['
	                  printf '%s\n' '    {'
	                  printf '%s\n' '      "Expiration": {'
	                  printf '        "Days": %s\n' "\$days"
	                  printf '%s\n' '      },'
	                  printf '      "ID": "monadiam-expire-after-%s-days",\n' "\$days"
	                  printf '%s\n' '      "Filter": {'
	                  printf '%s\n' '        "Prefix": ""'
	                  printf '%s\n' '      },'
	                  printf '%s\n' '      "Status": "Enabled"'
	                  printf '%s\n' '    }'
	                  printf '%s\n' '  ]'
	                  printf '%s\n' '}'
	              } >"/tmp/\$bucket-lifecycle.json"
	              mc ilm rule import "local/\$bucket" <"/tmp/\$bucket-lifecycle.json"
	              mc retention info "local/\$bucket"
	              mc ilm rule ls "local/\$bucket"
	          }
	
	          configure_archive_bucket "\$MINIO_AUDIT_LOG_BUCKET" "\$MINIO_AUDIT_LOG_RETENTION_DAYS"
	          configure_archive_bucket "\$MINIO_CHANGE_LOG_BUCKET" "\$MINIO_CHANGE_LOG_RETENTION_DAYS"
	
	          {
	              printf '%s\n' '{'
	              printf '%s\n' '  "Version": "2012-10-17",'
	              printf '%s\n' '  "Statement": ['
	              printf '%s\n' '    {'
	              printf '%s\n' '      "Effect": "Allow",'
	              printf '%s\n' '      "Action": ['
	              printf '%s\n' '        "s3:PutObject",'
	              printf '%s\n' '        "s3:GetObject"'
	              printf '%s\n' '      ],'
	              printf '%s\n' '      "Resource": ['
	              printf '        "arn:aws:s3:::%s/*",\n' "\$MINIO_AUDIT_LOG_BUCKET"
	              printf '        "arn:aws:s3:::%s/*"\n' "\$MINIO_CHANGE_LOG_BUCKET"
	              printf '%s\n' '      ]'
	              printf '%s\n' '    }'
	              printf '%s\n' '  ]'
	              printf '%s\n' '}'
	          } >/tmp/archiver-policy.json
	
	          mc admin policy create local archiver-policy /tmp/archiver-policy.json
	          mc admin user add local "\$MINIO_ARCHIVER_ACCESS_KEY" "\$MINIO_ARCHIVER_SECRET_KEY" >/dev/null 2>&1 || true
	          mc admin policy attach local archiver-policy --user "\$MINIO_ARCHIVER_ACCESS_KEY"
	          mc admin policy info local archiver-policy >/dev/null
	      volumeMounts:
	        - name: minio-main-root
	          mountPath: /mnt/minio-root
	          readOnly: true
	        - name: minio-main-archiver
	          mountPath: /mnt/minio-archiver
	          readOnly: true
	        - name: minio-main-tls
	          mountPath: /mnt/minio-tls
	          readOnly: true
	      securityContext:
	        allowPrivilegeEscalation: false
	        capabilities:
	          drop:
	            - ALL
	        runAsNonRoot: true
	        runAsUser: 1000
	        runAsGroup: 1000
	        seccompProfile:
	          type: RuntimeDefault
	  volumes:
	    - name: minio-main-root
	      secret:
	        secretName: minio-main-root
	    - name: minio-main-archiver
	      secret:
	        secretName: minio-main-archiver
	    - name: minio-main-tls
	      secret:
	        secretName: minio-main-tls
	EOF

    wait_minio_bootstrap_pod_completed "$namespace" "$name"
}
