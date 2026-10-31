# Code guide

Coding conventions and examples for TypeScript application code using event-driven architecture (EDA).

**Out of scope:** persistence, distributed delivery, and toolchain versions.

## Organization

### Namespace boundaries

Each module **must** export one named namespace containing its public classes and interfaces. Namespace and class names **must** use PascalCase; methods and parameters **must** use camelCase. Imports **must** remain module-scoped; global namespace merging is prohibited.

The compiler **must** emit transformed JavaScript; `erasableSyntaxOnly` is prohibited because namespaces require transformation.

### Class behavior

Application operations, event handlers, and adapters **must** be classes with instance methods and explicit return types. Event payloads **must** be readonly class properties. Free-standing function declarations are prohibited; inline callbacks **must** only forward arguments to methods. The composition entry point **must** be a static method.

Illustration — `src/notifications/order-log.ts`:

```typescript
import type { OrderEvents } from "../orders/events.js";

export namespace Notifications {
  export class OrderLog {
    handle(event: OrderEvents.OrderPlaced): void {
      console.log(`Order placed: ${event.orderId}`);
    }
  }
}
```

## Events

### Local event delivery

Publishers **must** describe a fact with an event class and **must not** import subscribers. The local adapter **must** use Node.js `node:events` to dispatch events synchronously in subscription order. Subscribers **must not** return promises. With no subscribers, publication **must** do nothing; the first thrown handler error **must** propagate and stop remaining handlers. Delivery **must** remain in-process, without persistence or retries.

Illustration — an already recorded order is announced; `src/orders/events.ts`:

```typescript
export namespace OrderEvents {
  export class OrderPlaced {
    constructor(public readonly orderId: string) {}
  }
}
```

Illustration — `src/messaging/local-events.ts`:

```typescript
import { EventEmitter } from "node:events";
import type { OrderEvents } from "../orders/events.js";
import type { Orders } from "../orders/announce-order-placed.js";

export namespace Messaging {
  export class LocalEvents implements Orders.EventPublisher {
    private readonly emitter = new EventEmitter();
    private readonly topic = "order.placed";

    subscribe(handler: (event: OrderEvents.OrderPlaced) => void): void {
      this.emitter.on(this.topic, handler);
    }

    publish(event: OrderEvents.OrderPlaced): void {
      this.emitter.emit(this.topic, event);
    }
  }
}
```

### Consumer-owned abstraction

Under dependency inversion (DIP), the application **must** own its event publisher interface; the adapter **must** implement it. The application **must not** import the adapter or `node:events`. The indirection's pain trigger is an anti-corruption boundary isolating the application's event contract from Node.js dispatch APIs.

Illustration — `src/orders/announce-order-placed.ts`:

```typescript
import { OrderEvents } from "./events.js";

export namespace Orders {
  export interface EventPublisher {
    publish(event: OrderEvents.OrderPlaced): void;
  }

  export class AnnounceOrderPlaced {
    constructor(private readonly events: EventPublisher) {}

    execute(orderId: string): void {
      this.events.publish(new OrderEvents.OrderPlaced(orderId));
    }
  }
}
```

## Composition

### Constructor injection

Under dependency injection (DI), the application **must** receive its publisher through the constructor. The concrete publisher, application operation, and subscriber **must** be instantiated and connected only in the composition entry point. Service locators and dependency injection containers are prohibited.

Illustration — `src/main.ts`; expected output: `Order placed: order-1`:

```typescript
import { Orders } from "./orders/announce-order-placed.js";
import { Messaging } from "./messaging/local-events.js";
import { Notifications } from "./notifications/order-log.js";

export namespace Composition {
  export class Application {
    static run(): void {
      const events = new Messaging.LocalEvents();
      const log = new Notifications.OrderLog();
      events.subscribe(event => log.handle(event));

      const announce = new Orders.AnnounceOrderPlaced(events);
      announce.execute("order-1");
    }
  }
}

Composition.Application.run();
```
