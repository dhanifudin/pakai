import QtQuick
import qs.Common
import qs.Modules.Plugins
import qs.Widgets

PluginSettings {
    id: root
    pluginId: "pakai"

    StyledText {
        width: parent.width
        text: "PakAI Settings"
        font.pixelSize: Theme.fontSizeLarge
        font.weight: Font.Bold
        color: Theme.surfaceText
    }

    StyledText {
        width: parent.width
        text: "The widget reads PakAI's local daemon. Theme colors follow DankMaterialShell automatically."
        font.pixelSize: Theme.fontSizeSmall
        color: Theme.surfaceVariantText
        wrapMode: Text.WordWrap
    }

    StringSetting {
        settingKey: "daemonUrl"
        label: "Daemon URL"
        description: "PakAI HTTP daemon address"
        defaultValue: "http://127.0.0.1:7731"
    }

    SliderSetting {
        settingKey: "refreshInterval"
        label: "Cache refresh interval"
        description: "How often the widget reads the daemon cache"
        defaultValue: 60
        minimum: 15
        maximum: 300
        unit: "sec"
    }

    ToggleSetting {
        settingKey: "showLabel"
        label: "Show PakAI label"
        description: "Display the name next to usage in a horizontal bar"
        defaultValue: false
    }
}
