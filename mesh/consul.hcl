server = true
ui_config {
    enabled = true
}

bootstrap_expect = 1
bind_addr        = "{{ GetInterfaceIP \"eth0\" }}"
client_addr      = "0.0.0.0"
data_dir         = "/consul/data"

ports {
    grpc = 8502
}

connect {
    enabled = true
}
