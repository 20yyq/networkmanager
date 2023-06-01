/**
 * @Author: Eacher
 * @Date:   2023-05-22 10:42:36
 * @LastEditTime:   2023-06-01 15:43:11
 * @LastEditors:    Eacher
 * --------------------------------------------------------------------------------------------------------------------<
 * @Description:  gcc 编译动态库： gcc -shared -fPIC -o libgonmcli.so network.c `pkg-config --cflags --libs libnm`
 *      gcc 编译静态库： gcc -c -fPIC -o gonmcli.o network.c `pkg-config --cflags --libs libnm`  \
 *      ar rcs libgonmcli.a gonmcli.o
 * --------------------------------------------------------------------------------------------------------------------<
 * @FilePath:   /gonmcli/network.c
 */
#include "network.h"

GMainLoop   *loop   = NULL;
NMClient    *client = NULL;
PermissionData Permission = {
    .ednetwork = 0, .ednwifi = 0, .edwwan = 0, .edwimax = 0, .sleep_wake = 0, .network_control = 0,
    .wifi_protected = 0, .wifi_open = 0, .modify_system = 0, .modify_own = 0, .modify_hostname = 0,
    .modify_dns = 0, .reload = 0, .checkpoint = 0, .edstatic = 0, .connectivity_check = 0,
};

/********************************** Start **********************************/
const char *getDeviceType(NMDevice *device);
const char *getDeviceState(NMDevice *device);
void swapConnection(NMConnection *conn, ConnData *data);
void swapDevice(NMDevice *device, DevData *data);
int checkClientPermission(NMClientPermission p);

/********************************** End **********************************/

gboolean init() {
    if (NULL != client)
        return FALSE;
    GError *error = NULL;
    loop = g_main_loop_new(NULL, FALSE);
    if (!(client = nm_client_new(NULL, &error))) {
        g_error_free(error);
        return FALSE;
    }
    if (!nm_client_get_nm_running(client)) {
        g_object_unref(client);
        return FALSE;
    }
    return TRUE;
}

// GoLang 实现该回调函数
extern void setConnectionFunc(ConnData *data);
extern void setDeviceFunc(DevData *data);

void runLoop() {
    ConnData cd;
    const GPtrArray *connections = nm_client_get_connections(client);
    for (int i = 0; i < connections->len; i++) {
        swapConnection(connections->pdata[i], &cd);
        setConnectionFunc(&cd);
    }
    DevData dd;
    const GPtrArray *devices = nm_client_get_devices(client);
    for (int i = 0; i < devices->len; i++) {
        swapDevice(devices->pdata[i], &dd);
        setDeviceFunc(&dd);
    }
    Permission.ednetwork    = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_NETWORK);
    Permission.ednwifi      = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_WIFI);
    Permission.edwwan       = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_WWAN);
    Permission.edwimax      = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_WIMAX);
    Permission.sleep_wake   = checkClientPermission(NM_CLIENT_PERMISSION_SLEEP_WAKE);
    Permission.network_control= checkClientPermission(NM_CLIENT_PERMISSION_NETWORK_CONTROL);
    Permission.wifi_protected= checkClientPermission(NM_CLIENT_PERMISSION_WIFI_SHARE_PROTECTED);
    Permission.wifi_open    = checkClientPermission(NM_CLIENT_PERMISSION_WIFI_SHARE_OPEN);
    Permission.modify_system= checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_SYSTEM);
    Permission.modify_own   = checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_OWN);
    Permission.modify_hostname= checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_HOSTNAME);
    Permission.modify_dns   = checkClientPermission(NM_CLIENT_PERMISSION_SETTINGS_MODIFY_GLOBAL_DNS);
    Permission.reload       = checkClientPermission(NM_CLIENT_PERMISSION_RELOAD);
    Permission.checkpoint   = checkClientPermission(NM_CLIENT_PERMISSION_CHECKPOINT_ROLLBACK);
    Permission.edstatic     = checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_STATISTICS);
    Permission.connectivity_check= checkClientPermission(NM_CLIENT_PERMISSION_ENABLE_DISABLE_CONNECTIVITY_CHECK);
    g_main_loop_run(loop);
    g_main_loop_unref(loop);
    g_object_unref(client);
    loop = NULL;
    client = NULL;
}

void quitLoop() {
    g_main_loop_quit(loop);
}

int checkClientPermission(NMClientPermission p) {
    NMClientPermissionResult r = nm_client_get_permission_result(client, p);
    return r == NM_CLIENT_PERMISSION_RESULT_YES ? 1 : 0;
}

void swapConnection(NMConnection *conn, ConnData *data) {
    data->_type             = nm_connection_get_connection_type(conn);
    data->dbus_path         = nm_connection_get_path(conn);
    data->firmware          = "";
    NMSettingIPConfig *ipConfig = nm_connection_get_setting_ip4_config(conn);
    if (ipConfig) {
        data->ipv4_method   = nm_setting_ip_config_get_method(ipConfig);
        data->ipv4_dns      = nm_setting_ip_config_get_num_dns(ipConfig) ?
                                nm_setting_ip_config_get_dns(ipConfig, 0) : "nil";
        data->ipv4_addresses= nm_setting_ip_config_get_num_addresses(ipConfig) ?
                                nm_ip_address_get_address(nm_setting_ip_config_get_address(ipConfig, 0)) : "nil";
        data->ipv4_gateway  = nm_setting_ip_config_get_gateway(ipConfig);
    }
    NMSettingConnection *setting = nm_connection_get_setting_connection(conn);
    if (setting) {
        data->id            = nm_setting_connection_get_id(setting);
        data->uuid          = nm_setting_connection_get_uuid(setting);
        data->priority      = nm_setting_connection_get_autoconnect_priority(setting);
        data->autoconnect   = nm_setting_connection_get_autoconnect(setting);
    }
}

void swapDevice(NMDevice *device, DevData *data) {
    data->iface = nm_device_get_ip_iface(device);
    data->_type = getDeviceType(device);
    data->udi = nm_device_get_udi(device);
    data->driver = nm_device_get_driver(device);
    data->firmware = nm_device_get_firmware_version(device);
    data->hw_address = nm_device_get_hw_address(device);
    data->state = getDeviceState(device);
    NMActiveConnection *ac = nm_device_get_active_connection(device);
    data->uuid = "";
    if (ac) data->uuid = nm_active_connection_get_uuid(ac);
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

/*************************************************************** WIFI Start *********************************************************************/

// GoLang 实现该回调函数
extern int scanCallBackFunc(const char *name, guint n, WifiData *wd);
void wifiScanCallback(GObject* object, GAsyncResult* res, gpointer data);
void wifiScanCallback(GObject* object, GAsyncResult* res, gpointer data) {
    if (NULL != res) {
        const GPtrArray *list = nm_device_wifi_get_access_points(NM_DEVICE_WIFI(g_async_result_get_source_object(res)));
        if (1 == scanCallBackFunc("start", list->len, NULL)) {
            WifiData wd;
            for (int i = 0; i < list->len; i++) {
                NM80211Mode mode    = nm_access_point_get_mode(list->pdata[i]);
                GBytes *ssid        = nm_access_point_get_ssid(list->pdata[i]);
                if (ssid)
                    wd.ssid    = nm_utils_ssid_to_utf8(g_bytes_get_data(ssid, NULL), g_bytes_get_size(ssid));
                wd.freq        = nm_access_point_get_frequency(list->pdata[i]);
                wd.bitrate     = nm_access_point_get_max_bitrate(list->pdata[i]);
                wd.flags       = nm_access_point_get_flags(list->pdata[i]);
                wd.strength    = nm_access_point_get_strength(list->pdata[i]);
                wd.bssid       = nm_access_point_get_bssid(list->pdata[i]);
                wd.mode        = mode == NM_802_11_MODE_ADHOC   ? "Ad-Hoc" :
                                mode == NM_802_11_MODE_INFRA ? "Infrastructure" : "Unknown";
                wd.dbus_path   = nm_object_get_path(NM_OBJECT(list->pdata[i]));
                scanCallBackFunc("runFunc", (guint)i, &wd);
                if (NULL != wd.ssid) {
                    g_free(wd.ssid);
                    wd.ssid = NULL;
                }
            }
            scanCallBackFunc("close", 0, NULL);
        }
    }
}

int wifiScanAsync() {
    NMDevice *device = NULL;
    const GPtrArray *devices = nm_client_get_devices(client);
    for (int i = 0; i < devices->len; i++) {
        device = g_ptr_array_index(devices, i);
        if (NM_IS_DEVICE_WIFI(device))
            break;
        device = NULL;
    }
    if (NULL == device)
        return 0;
    nm_device_wifi_request_scan_options_async(NM_DEVICE_WIFI(device), NULL, NULL, wifiScanCallback, NULL);
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
    NMDevice *dev = nm_client_get_device_by_iface(client, iface);
    if (NULL != dev) {
        g_signal_handlers_disconnect_by_func(dev, device_state, client);
        g_signal_handlers_disconnect_by_func(dev, device_ac, client); 
    }
}

int notifyDeviceMonitor(const char *iface, char **type, char **bssid, char **connId) {
    NMDevice *dev = nm_client_get_device_by_iface(client, iface);
    if (NULL == dev)
        return 0;
    *type = g_strdup(getDeviceType(dev));
    *bssid = g_strdup(nm_device_get_hw_address(dev));
    *connId = g_strdup(nm_active_connection_get_id(nm_device_get_active_connection(dev)));
    g_signal_connect(dev, "notify::" "state", G_CALLBACK(device_state), client);
    g_signal_connect(dev, "notify::" "active-connection", G_CALLBACK(device_ac), client);
    return 1;
}


/************************************************************* Device End *******************************************************************/

