import { notifications, NotificationData } from '@mantine/notifications';

export function showNotificationWithClean(data: NotificationData) {
  notifications.clean();
  notifications.show(data);
}
