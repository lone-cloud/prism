# The Privacy Paradox of Self-Hosting

## Preface

A self-proclaimed infosec nerd blew my mind with this realization on the Graphene OS forums. Since then I've stopped self-hosting my mollysocket, ntfy, Vaultwarden and FMD servers. I ended up switching back to Bitwarden and also I'm now using Molly.im with WebSockets for push notifications. I realize that self-hosting is often talked about as a great way to protect your privacy and fully own your data (and metadata). I thought this would be fun to argue that you should almost never self-host anything and especially if you intend to expose and access it on the internet.

## Network footprinting

As we use our devices we leave a digital footprint on the network. This footprint is composed of the requests that we make to the digital services that we interact with. Sure, there are some ways to muddy this footprint by using Tor and/or VPNs.

There are very definite drawbacks to using such tools as Tor is dreadfully slow, which makes it very painful for everyday use especially for things like push notifications which one would expect to work quickly. VPNs shift the trust to your provider who is another party that you will need to trust to not invade your privacy and one that you will likely need to pay.

Using either of these options can make you a target for deeper surveillance as neither of these options will fully mask your use of them. A network sniffing adversary might start to wonder what you're trying to hide by needing to use such niche tools to muddy your network footprint.

## The network footprint of self-hosted systems

Unless you enjoy typing out IP addresses, you will likely need to register your self-hosted system to a domain which will be registered with the DNS. This domain may very well be linked back to you either through your payment or through the registrar's KYC system.

Even if you somehow don't use a domain name, you will be broadcasting your connection to a unique IP address that a network sniffer could pick up on. If that unique connection ever gets linked to the real you, like through your domain registration or other means, an adversary could track your location as you move around and connect to different networks like WiFi at cafes, libraries, airports, etc…

## My experience with degoogling and using UnifiedPush

I am fully degoogled meaning I run an Android OS (Graphene OS) without the Google Play services. The biggest challenge here is that Google Play services bundle in their Firebase Cloud Messaging, which you won't have resulting in the lack of push notifications for your apps. The community solution for this is to use the UnifiedPush system which requires you to download an UP distributor app like ntfy, which will maintain a single WebSocket connection to an UP server that will forward to you any notifications that you may get from any of the supporting apps. One such app that I frequently use is Molly which is a fork of Signal with great support for degoogled users like myself. Molly supports UP, FCM and direct WS connections to the Signal network.

As I started tracking the connections that my phone makes, I noticed that I'm constantly making connections to a super niche UP server to await for push notifications. These connections are a technical requirement as WS must routinely confirm the client's connection by sending out heartbeat messages. In my case I was also self-hosting a private ntfy server (together with a Molly-required MollySocket server) that was served from my private domain. I realized that this is very poor privacy hygiene as anything that was sniffing my network could easily figure out that it's my requests based on the very unique URL.

## My solution: SUP bro

Signal UnifiedPush will consist of an Android app that will act as an UP distributor. It will listen on Signal notifications to potentially act on them like to wake an app to refresh its data.

SUP will also consist of a strictly self-hostable server component that will proxy the UP push notifications through Signal groups instead of through an UP server like ntfy.sh. This server must be self-hosted as it will be linked as a new device to the user's Signal account to receive the notifications. After the initial setup, the server will not need to be exposed to the internet as it will only make outbound connections to the Signal servers.

## Closing thoughts

Self-hosting is still valuable for many use cases, especially when services don't need to be accessed over the internet. However, when it comes to services you access remotely, the privacy trade-offs are often worse than using established providers that blend your traffic with millions of other users.

The SUP project aims to provide a middle ground: the privacy benefits of using a widely-adopted service (Signal) while maintaining the control and ownership that self-hosting provides.

