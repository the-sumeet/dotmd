//go:build ios

// DotmdSceneDelegate adopts the UIScene life cycle on top of Wails' app
// delegate.
//
// Apps linked against the iOS 27 SDK must use UIScene, or UIKit refuses to
// launch them ("UIScene life cycle is required for apps built with this SDK").
// Wails (checked: v3.0.0-beta.19 and v3.1.0) only implements a classic
// UIApplicationDelegate that builds its UIWindow in
// didFinishLaunchingWithOptions. That runs before any scene connects, so here
// we attach that existing window to the scene, and forward the scene life
// cycle callbacks (which replace the app-level ones under UIScene) to the
// Wails delegate so its application events keep firing.
//
// Info.plist must name this class under UIApplicationSceneManifest
// (build/ios/Taskfile.yml: patch:scene-manifest).
// Drop this file once Wails adopts UIScene itself.

#import <UIKit/UIKit.h>

@interface DotmdSceneDelegate : UIResponder <UIWindowSceneDelegate>
@property (strong, nonatomic) UIWindow *window;
// UIKit sends sceneWillEnterForeground on cold launch too, where the classic
// app delegate never got applicationWillEnterForeground. Only forward it
// after a real trip to the background.
@property (nonatomic) BOOL wasInBackground;
@end

@implementation DotmdSceneDelegate

// Calls an optional UIApplicationDelegate callback (e.g.
// applicationDidBecomeActive:) on the Wails delegate, if it implements it.
- (void)forwardToAppDelegate:(SEL)selector {
    UIApplication *app = UIApplication.sharedApplication;
    id<UIApplicationDelegate> delegate = app.delegate;
    if ([delegate respondsToSelector:selector]) {
        ((void (*)(id, SEL, UIApplication *))[(id)delegate methodForSelector:selector])(delegate, selector, app);
    }
}

- (void)scene:(UIScene *)scene
    willConnectToSession:(UISceneSession *)session
                 options:(UISceneConnectionOptions *)connectionOptions {
    if (![scene isKindOfClass:[UIWindowScene class]]) {
        return;
    }
    UIWindowScene *windowScene = (UIWindowScene *)scene;
    id<UIApplicationDelegate> delegate = UIApplication.sharedApplication.delegate;
    UIWindow *window = [delegate respondsToSelector:@selector(window)] ? delegate.window : nil;
    if (window == nil) {
        // Wails builds its window synchronously in didFinishLaunching, which
        // UIKit always runs before connecting the first scene. If that ever
        // changes, fail loudly instead of showing a silent black screen.
        NSLog(@"DotmdSceneDelegate: Wails app delegate has no window to attach; "
              @"the Wails iOS launch sequence has changed - update internal/iosscene.");
        return;
    }
    window.windowScene = windowScene;
    window.frame = windowScene.coordinateSpace.bounds;
    self.window = window;
    [window makeKeyAndVisible];
}

- (void)sceneDidBecomeActive:(UIScene *)scene {
    [self forwardToAppDelegate:@selector(applicationDidBecomeActive:)];
}

- (void)sceneWillResignActive:(UIScene *)scene {
    [self forwardToAppDelegate:@selector(applicationWillResignActive:)];
}

- (void)sceneWillEnterForeground:(UIScene *)scene {
    if (!self.wasInBackground) {
        return;
    }
    self.wasInBackground = NO;
    [self forwardToAppDelegate:@selector(applicationWillEnterForeground:)];
}

- (void)sceneDidEnterBackground:(UIScene *)scene {
    self.wasInBackground = YES;
    [self forwardToAppDelegate:@selector(applicationDidEnterBackground:)];
}

@end
