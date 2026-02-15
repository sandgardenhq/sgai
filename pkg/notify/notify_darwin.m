#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

static BOOL authorizationRequested = NO;
static BOOL authorizationGranted = NO;

static void requestAuthorization(void) {
	if (authorizationRequested) {
		return;
	}
	authorizationRequested = YES;

	if ([[NSBundle mainBundle] bundleIdentifier] == nil) {
		return;
	}

	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	dispatch_semaphore_t sem = dispatch_semaphore_create(0);
	[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
		completionHandler:^(BOOL granted, NSError *error) {
			authorizationGranted = granted;
			dispatch_semaphore_signal(sem);
		}];
	dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 5 * NSEC_PER_SEC));
}

static void sendWithUNNotification(NSString *title, NSString *message) {
	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
	content.title = title;
	content.body = message;
	content.sound = [UNNotificationSound defaultSound];

	NSString *identifier = [[NSUUID UUID] UUIDString];
	UNNotificationRequest *request = [UNNotificationRequest
		requestWithIdentifier:identifier
		content:content
		trigger:nil];
	[center addNotificationRequest:request withCompletionHandler:nil];
}

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
static void sendWithNSUserNotification(NSString *title, NSString *message) {
	NSUserNotification *notification = [[NSUserNotification alloc] init];
	notification.title = title;
	notification.informativeText = message;
	notification.soundName = NSUserNotificationDefaultSoundName;
	[[NSUserNotificationCenter defaultUserNotificationCenter] deliverNotification:notification];
}
#pragma clang diagnostic pop

void SendNativeNotification(const char *title, const char *message) {
	NSString *nsTitle = [NSString stringWithUTF8String:title];
	NSString *nsMessage = [NSString stringWithUTF8String:message];

	requestAuthorization();

	if (authorizationGranted) {
		sendWithUNNotification(nsTitle, nsMessage);
	} else {
		sendWithNSUserNotification(nsTitle, nsMessage);
	}
}
