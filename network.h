#ifndef __go_network_h
#define __go_network_h
#include <NetworkManager.h>

typedef struct {
    guint32     flags;
    guint32     freq;
    guint32     bitrate;
    guint8      strength;
    char *ssid;
    const char *bssid;
    const char *mode;
    const char *dbus_path;
} WifiData;

gboolean init();
void runLoop();
void quitLoop();

// WIIF func
int wifiScanAsync();

// Devices func
int notifyDeviceMonitor(const char *iface, char **type, char **bssid, char **connId);
void removeDeviceMonitor(const char *iface);

#endif /* __go_network_h */