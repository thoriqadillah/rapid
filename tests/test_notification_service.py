from __future__ import annotations

from typing import cast
from unittest.mock import MagicMock, call

import pytest

from PySide6.QtGui import QAction, QIcon
from PySide6.QtWidgets import QMenu, QSystemTrayIcon

from rapid.backend.notification import NotificationService
from rapid.qml.icons import icons_rc  # noqa: F401  registers qrc resources on import


@pytest.fixture
def trayIcon() -> MagicMock:
    return MagicMock(spec=QSystemTrayIcon)


@pytest.fixture
def trayMenu() -> tuple[MagicMock, list[MagicMock]]:
    menu = MagicMock(spec=QMenu)
    actions = [MagicMock(spec=QAction) for _ in range(3)]
    menu.addAction.side_effect = actions
    return menu, actions


@pytest.mark.parametrize("method, kind", [
    ("error", "error"),
    ("success", "success"),
    ("info", "info"),
])
def test_notification_methods_emit_only_in_app_by_default(
    trayIcon: MagicMock,
    trayMenu: tuple[MagicMock, list[MagicMock]],
    method: str,
    kind: str,
) -> None:
    menu, _ = trayMenu
    service = NotificationService(
        trayIcon=cast(QSystemTrayIcon, trayIcon),
        trayMenu=cast(QMenu, menu),
    )
    notifications: list[tuple[str, str, str]] = []
    service.notificationRequested.connect(
        lambda emitted_kind, title, message: notifications.append(
            (emitted_kind, title, message)
        )
    )

    getattr(service, method)("Title", "Message")

    assert notifications == [(kind, "Title", "Message")]
    trayIcon.showMessage.assert_not_called()


@pytest.mark.parametrize("method", ["error", "success", "info"])
def test_notification_methods_optionally_show_desktop_notification(
    trayIcon: MagicMock,
    trayMenu: tuple[MagicMock, list[MagicMock]],
    method: str,
) -> None:
    menu, _ = trayMenu
    notificationIcon = cast(QIcon, MagicMock(spec=QIcon))
    service = NotificationService(
        trayIcon=cast(QSystemTrayIcon, trayIcon),
        notificationIcon=notificationIcon,
        trayMenu=cast(QMenu, menu),
    )

    getattr(service, method)("Title", "Message", True)

    trayIcon.showMessage.assert_called_once_with(
        "Title", "Message", notificationIcon, 5000
    )


def test_tray_menu_actions_emit_requests(
    trayIcon: MagicMock,
    trayMenu: tuple[MagicMock, list[MagicMock]],
) -> None:
    menu, actions = trayMenu
    service = NotificationService(
        trayIcon=cast(QSystemTrayIcon, trayIcon),
        trayMenu=cast(QMenu, menu),
    )
    requests: list[str] = []
    service.openRequested.connect(lambda: requests.append("open"))
    service.newDownloadRequested.connect(lambda: requests.append("new-download"))
    service.quitRequested.connect(lambda: requests.append("quit"))

    for action in actions:
        action.triggered.connect.call_args.args[0]()

    menu.addAction.assert_has_calls([
        call("Open Rapid"),
        call("New download"),
        call("Quit"),
    ])
    trayIcon.setContextMenu.assert_called_once_with(menu)
    assert requests == ["open", "new-download", "quit"]


def test_close_releases_native_tray_resources(
    trayIcon: MagicMock,
    trayMenu: tuple[MagicMock, list[MagicMock]],
) -> None:
    menu, _ = trayMenu
    service = NotificationService(
        trayIcon=cast(QSystemTrayIcon, trayIcon),
        trayMenu=cast(QMenu, menu),
    )

    service.close()
    service.close()

    trayIcon.hide.assert_called_once_with()
    menu.close.assert_called_once_with()


def test_clicking_tray_icon_requests_open(
    trayIcon: MagicMock,
    trayMenu: tuple[MagicMock, list[MagicMock]],
) -> None:
    menu, _ = trayMenu
    service = NotificationService(
        trayIcon=cast(QSystemTrayIcon, trayIcon),
        trayMenu=cast(QMenu, menu),
    )
    requests: list[str] = []
    service.openRequested.connect(lambda: requests.append("open"))

    service._onTrayActivated(QSystemTrayIcon.ActivationReason.Trigger)
    service._onTrayActivated(QSystemTrayIcon.ActivationReason.Context)

    assert requests == ["open"]
