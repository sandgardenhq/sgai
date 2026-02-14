#import <Cocoa/Cocoa.h>

extern void goMenuItemClicked(int tag);

static NSStatusItem *statusItem = nil;
static NSMenu *statusMenu = nil;

@interface MenuBarDelegate : NSObject
- (void)menuItemClicked:(NSMenuItem *)sender;
@end

@implementation MenuBarDelegate
- (void)menuItemClicked:(NSMenuItem *)sender {
	int tag = (int)sender.tag;
	dispatch_async(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^{
		goMenuItemClicked(tag);
	});
}
@end

static MenuBarDelegate *menuDelegate = nil;

void MenuBarInit(void) {
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

	menuDelegate = [[MenuBarDelegate alloc] init];

	statusItem = [[NSStatusBar systemStatusBar]
		statusItemWithLength:NSVariableStatusItemLength];
	statusItem.button.title = @"\u25CB sgai";
	statusItem.button.toolTip = @"SGAI Factory Monitor";

	statusMenu = [[NSMenu alloc] init];
	statusItem.menu = statusMenu;
}

void MenuBarSetTitle(const char *title) {
	NSString *nsTitle = [NSString stringWithUTF8String:title];
	dispatch_async(dispatch_get_main_queue(), ^{
		if (statusItem != nil) {
			statusItem.button.title = nsTitle;
		}
	});
}

void MenuBarClear(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (statusMenu != nil) {
			[statusMenu removeAllItems];
		}
	});
}

void MenuBarAddItem(const char *title, int tag, int enabled) {
	NSString *nsTitle = [NSString stringWithUTF8String:title];
	int t = tag;
	int e = enabled;
	dispatch_async(dispatch_get_main_queue(), ^{
		if (statusMenu == nil || menuDelegate == nil) return;
		NSMenuItem *item = [[NSMenuItem alloc]
			initWithTitle:nsTitle
			action:@selector(menuItemClicked:)
			keyEquivalent:@""];
		item.target = menuDelegate;
		item.tag = t;
		item.enabled = e ? YES : NO;
		[statusMenu addItem:item];
	});
}

void MenuBarAddSeparator(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (statusMenu != nil) {
			[statusMenu addItem:[NSMenuItem separatorItem]];
		}
	});
}

void MenuBarOpenURL(const char *urlStr) {
	NSString *nsURL = [NSString stringWithUTF8String:urlStr];
	dispatch_async(dispatch_get_main_queue(), ^{
		NSURL *url = [NSURL URLWithString:nsURL];
		if (url != nil) {
			[[NSWorkspace sharedWorkspace] openURL:url];
		}
	});
}

void MenuBarRunLoop(void) {
	[NSApp run];
}

void MenuBarStop(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp terminate:nil];
	});
}
