/**
 * @Author: Eacher
 * @Date:   2023-05-22 10:42:36
 * @LastEditTime:   2023-06-03 16:04:47
 * @LastEditors:    Eacher
 * --------------------------------------------------------------------------------------------------------------------<
 * @Description:  gcc 编译动态库： gcc -shared -fPIC -o libgonmcli.so network.c `pkg-config --cflags --libs libnm`
 *      gcc 编译静态库： gcc -c -fPIC -o gonmcli.o network.c `pkg-config --cflags --libs libnm`  \
 *      ar rcs libgonmcli.a gonmcli.o
 * --------------------------------------------------------------------------------------------------------------------<
 * @FilePath:   /gonmcli/network.c
 */
#include "network.h"

clientData Client = {
    .loop = NULL, .client = NULL, .wifiData = NULL, .connData = NULL,
    .devData = NULL, .wifiDataLen = 0, .connDataLen = 0, .devDataLen = 0, 
    .permission = {
        .ednetwork = 0, .ednwifi = 0, .edwwan = 0, .edwimax = 0, .sleep_wake = 0, .network_control = 0,
        .wifi_protected = 0, .wifi_open = 0, .modify_system = 0, .modify_own = 0, .modify_hostname = 0,
        .modify_dns = 0, .reload = 0, .checkpoint = 0, .edstatic = 0, .connectivity_check = 0,
    },
};

WifiData *getWifiData(int i) {
    WifiData *o = NULL;
    if (Client.wifiDataLen > i)
        o = &Client.wifiData[i];
    return o;
}

ConnData *getConnData(int i) {
    ConnData *o = NULL;
    if (Client.connDataLen > i)
        o = &Client.connData[i];
    return o;
}

DevData *getDevData(int i) {
    DevData *o = NULL;
    if (Client.devDataLen > i)
        o = &Client.devData[i];
    return o;
}

// GoLang 实现该回调函数
extern void initCallBackFunc();
const char *getDeviceType(NMDevice *device);
const char *getDeviceState(NMDevice *device);
int checkClientPermission(NMClientPermission p);
void swapConnection(NMConnection *conn, ConnData *data);
void swapDevice(NMDevice *device, DevData *data);
void init() {
    const GPtrArray *connections = nm_client_get_connections(Client.client);
    if (0 < connections->len) {
        do {
            Client.connData = (ConnData *)g_malloc(sizeof(ConnData) * connections->len);
        } while(Client.connData == NULL);
        Client.connDataLen = (uint)connections->len;
        for (int i = 0; i < connections->len; i++) {
            swapConnection(connections->pdata[i], &Client.connData[i]);
        }
    }
    const GPtrArray *devices = nm_client_get_devices(Client.client);
    if (0 < devices->len) {
        do {
            Client.devData = (DevData *)g_malloc(sizeof(DevData) * devices->len);
        } while(Client.devData == NULL);
        Client.devDataLen = (uint)devices->len;
        for (int i = 0; i < devices->len; i++) {
            swapDevice(devices->pdata[i], &Client.devData[i]);
        }
    }
    Client.permission.ednetwork    = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_NETWORK);
    Client.permission.ednwifi      = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_WIFI);
    Client.permission.edwwan       = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_WWAN);
    Client.permission.edwimax      = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_WIMAX);
    Client.permission.sleep_wake   = checkClientPermission(NM_CLIENT_PERMISSION_SLEEP_WAKE);
    Client.permission.network_control= checkClientPermission(NM_CLIENT_PERMISSION_NETWORK_CONTROL);
    Client.permission.wifi_protected= checkClientPermission(NM_CLIENT_PERMISSION_WIFI_SHARE_PROTECTED);
    Client.permission.wifi_open    = checkClientPermission(NM_CLIENT_PERMISSION_WIFI_SHARE_OPEN);
    Client.permission.modify_system= checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_SYSTEM);
    Client.permission.modify_own   = checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_OWN);
    Client.permission.modify_hostname= checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_HOSTNAME);
    Client.permission.modify_dns   = checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_GLOBAL_DNS);
    Client.permission.reload       = checkClientPermission(NM_CLIENT_PERMISSION_RELOAD);
    Client.permission.checkpoint   = checkClientPermission(NM_CLIENT_PERMISSION_CHECKPOINT_ROLLBACK);
    Client.permission.edstatic     = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_STATISTICS);
    Client.permission.connectivity_check= checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_CONNECTIVITY_CHECK);
    initCallBackFunc();
    g_main_loop_run(Client.loop);
    g_main_loop_unref(Client.loop);
    g_object_unref(Client.client);
    if (Client.connData != NULL) {
        g_free(Client.connData);
        Client.connData = NULL;
    }
    if (Client.devData != NULL) {
        g_free(Client.devData);
        Client.devData = NULL;
    }
    if (Client.wifiData != NULL) {
        g_free(Client.wifiData);
        Client.wifiData = NULL;
    }
    Client.loop = NULL;
    Client.client = NULL;
}

void swapConnection(NMConnection *conn, ConnData *data) {
    data->dev           = NULL;
    data->_type         = nm_connection_get_connection_type(conn);
    data->dbus_path     = nm_connection_get_path(conn);
    data->firmware      = "";
    NMSettingIPConfig *ipConfig = nm_connection_get_setting_ip4_config(conn);
    data->ipv4_method   = nm_setting_ip_config_get_method(ipConfig);
    data->ipv4_dns      = nm_setting_ip_config_get_num_dns(ipConfig) ?
                            nm_setting_ip_config_get_dns(ipConfig, 0) : "nil";
    data->ipv4_addresses= nm_setting_ip_config_get_num_addresses(ipConfig) ?
                            nm_ip_address_get_address(nm_setting_ip_config_get_address(ipConfig, 0)) : "nil";
    data->ipv4_gateway  = nm_setting_ip_config_get_gateway(ipConfig);
    NMSettingConnection *setting = nm_connection_get_setting_connection(conn);
    data->id            = nm_setting_connection_get_id(setting);
    data->uuid          = nm_setting_connection_get_uuid(setting);
    data->priority      = nm_setting_connection_get_autoconnect_priority(setting);
    data->autoconnect   = nm_setting_connection_get_autoconnect(setting);
}

void swapDevice(NMDevice *device, DevData *data) {
    data->conn = NULL;
    data->iface = nm_device_get_ip_iface(device);
    data->_type = getDeviceType(device);
    data->udi = nm_device_get_udi(device);
    data->driver = nm_device_get_driver(device);
    data->firmware = nm_device_get_firmware_version(device);
    data->hw_address = nm_device_get_hw_address(device);
    data->state = getDeviceState(device);
    NMActiveConnection *ac = nm_device_get_active_connection(device);
    if (ac) {
        const char *uuid = nm_active_connection_get_uuid(ac);
        for (int i = 0; i < Client.connDataLen; i++) {
            if (uuid == Client.connData[i].uuid) {
                Client.connData[i].dev = data;
                data->conn = &Client.connData[i];
            }
        }
    }
    data->autoconnect = nm_device_get_autoconnect(device);
    data->real = nm_device_is_real(device);
    data->software = nm_device_is_software(device);
}

const char *getDeviceType(NMDevice *device) {
    const char *type;
    switch (nm_device_get_device_type(device)) {
    case NM_DEVICE_TYPE_ETHERNET:
        type = "ethernet";
        break;
    case NM_DEVICE_TYPE_WIFI:
        type = "wifi";
        break;
    case NM_DEVICE_TYPE_MODEM:
        type = "modem";
        break;
    case NM_DEVICE_TYPE_BOND:
        type = "bond";
        break;
    case NM_DEVICE_TYPE_VLAN:
        type = "vlan";
        break;
    case NM_DEVICE_TYPE_BRIDGE:
        type = "bridge";
        break;
    case NM_DEVICE_TYPE_GENERIC:
        type = "generic";
        break;
    default:
        type = "unknown";
        break;
    }
    return type;
}

const char *getDeviceState(NMDevice *device) {
    const char *state;
    switch (nm_device_get_state(device)) {
    case NM_DEVICE_STATE_UNMANAGED:
        state = "unmanaged";
        break;
    case NM_DEVICE_STATE_UNAVAILABLE:
        state = "unavailable";
        break;
    case NM_DEVICE_STATE_DISCONNECTED:
        state = "disconnected";
        break;
    case NM_DEVICE_STATE_PREPARE:
        state = "prepare";
        break;
    case NM_DEVICE_STATE_CONFIG:
        state = "config";
        break;
    case NM_DEVICE_STATE_NEED_AUTH:
        state = "need auth";
        break;
    case NM_DEVICE_STATE_IP_CONFIG:
        state = "ip config";
        break;
    case NM_DEVICE_STATE_IP_CHECK:
        state = "ip check";
        break;
    case NM_DEVICE_STATE_SECONDARIES:
        state = "secondaries";
        break;
    case NM_DEVICE_STATE_ACTIVATED:
        state = "activated";
        break;
    case NM_DEVICE_STATE_DEACTIVATING:
        state = "deactivating";
        break;
    case NM_DEVICE_STATE_FAILED:
        state = "failed";
        break;
    case NM_DEVICE_STATE_UNKNOWN:
    default:
        state = "unknown";
        break;
    }
    return state;
}

int checkClientPermission(NMClientPermission p) {
    NMClientPermissionResult r = nm_client_get_permission_result(Client.client, p);
    return r == NM_CLIENT_PERMISSION_RESULT_YES ? 1 : 0;
}

/*************************************************************** WIFI Start *********************************************************************/

// GoLang 实现该回调函数
extern void scanCallBackFunc(int idx);
void wifiScanCallback(GObject *object, GAsyncResult *res, gpointer data);
void wifiScanCallback(GObject *object, GAsyncResult *res, gpointer data) {
    int *idx = (int *)data;
    if (NULL != res) {
        const GPtrArray *list = nm_device_wifi_get_access_points(NM_DEVICE_WIFI(g_async_result_get_source_object(res)));
        if (1 < list->len) {
            do {
                if (Client.wifiData != NULL) {
                    g_free(Client.wifiData);
                    Client.wifiData = NULL;
                    Client.wifiDataLen = 0;
                }
                Client.wifiData = (WifiData *)g_malloc(sizeof(WifiData) * list->len);
            } while(Client.wifiData == NULL);
            Client.wifiDataLen = (uint)list->len;
            for (int i = 0; i < list->len; i++) {
                NM80211Mode mode    = nm_access_point_get_mode(list->pdata[i]);
                GBytes *ssid        = nm_access_point_get_ssid(list->pdata[i]);
                if (ssid)
                    Client.wifiData[i].ssid    = nm_utils_ssid_to_utf8(g_bytes_get_data(ssid, NULL), g_bytes_get_size(ssid));
                Client.wifiData[i].freq        = nm_access_point_get_frequency(list->pdata[i]);
                Client.wifiData[i].bitrate     = nm_access_point_get_max_bitrate(list->pdata[i]);
                Client.wifiData[i].flags       = nm_access_point_get_flags(list->pdata[i]);
                Client.wifiData[i].strength    = nm_access_point_get_strength(list->pdata[i]);
                Client.wifiData[i].bssid       = nm_access_point_get_bssid(list->pdata[i]);
                Client.wifiData[i].mode        = mode == NM_802_11_MODE_ADHOC   ? "Ad-Hoc" :
                                mode == NM_802_11_MODE_INFRA ? "Infrastructure" : "Unknown";
                Client.wifiData[i].dbus_path   = nm_object_get_path(NM_OBJECT(list->pdata[i]));
            }
        }
    }
    scanCallBackFunc(*idx);
    g_free(idx);
    idx = NULL;
}

int wifiScanAsync(int idx) {
    NMDevice *device = NULL;
    const GPtrArray *devices = nm_client_get_devices(Client.client);
    for (int i = 0; i < devices->len; i++) {
        device = g_ptr_array_index(devices, i);
        if (NM_IS_DEVICE_WIFI(device))
            break;
        device = NULL;
    }
    if (NULL == device)
        return 0;
    int *tmp = (int *)g_malloc(sizeof(int));
    *tmp = idx;
    nm_device_wifi_request_scan_options_async(NM_DEVICE_WIFI(device), NULL, NULL, wifiScanCallback, tmp);
    return 1;
}

/*************************************************************** WIFI End *********************************************************************/

/************************************************************* Device Start *******************************************************************/

// GoLang 实现该回调函数
extern void deviceMonitorCallBackFunc(const char *funcName, const char *devName, const char *state, guint n);
void device_state(NMDevice *device, GParamSpec *pspec, NMClient *client);
void device_state(NMDevice *device, GParamSpec *pspec, NMClient *client){
    deviceMonitorCallBackFunc("device_state", nm_device_get_iface(device), getDeviceState(device), (guint)nm_device_get_state(device));
}
void device_ac(NMDevice *device, GParamSpec *pspec, NMClient *client);
void device_ac(NMDevice *device, GParamSpec *pspec, NMClient *client) {
    deviceMonitorCallBackFunc("device_ac", nm_device_get_iface(device), getDeviceState(device), (guint)nm_device_get_state(device));
}

void removeDeviceMonitor(const char *iface) {
    NMDevice *dev = nm_client_get_device_by_iface(Client.client, iface);
    if (NULL != dev) {
        g_signal_handlers_disconnect_by_func(dev, device_state, Client.client);
        g_signal_handlers_disconnect_by_func(dev, device_ac, Client.client); 
    }
}

int notifyDeviceMonitor(const char *iface, char **type, char **bssid, char **connId) {
    NMDevice *dev = nm_client_get_device_by_iface(Client.client, iface);
    if (NULL == dev)
        return 0;
    *type = g_strdup(getDeviceType(dev));
    *bssid = g_strdup(nm_device_get_hw_address(dev));
    *connId = g_strdup(nm_active_connection_get_id(nm_device_get_active_connection(dev)));
    g_signal_connect(dev, "notify::" "state", G_CALLBACK(device_state), Client.client);
    g_signal_connect(dev, "notify::" "active-connection", G_CALLBACK(device_ac), Client.client);
    return 1;
}


/************************************************************* Device End *******************************************************************/

