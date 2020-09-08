FROM golang:1.14

COPY github-actions-exporter github-actions-exporter
RUN chmod 755 github-actions-exporter

CMD ["./github-actions-exporter"]
