apiVersion: v1
kind: Service
metadata:
  name: spire-server
  namespace: {{ .Release.Namespace }}
spec:
  type: NodePort
  ports:
    - name: grpc
      port: 8081
      targetPort: 8081
      protocol: TCP
  selector:
    app: spire-server
