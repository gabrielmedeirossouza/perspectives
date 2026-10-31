# TypeScript EDA example

Use this reference when a project requests event-driven architecture (EDA), mandatory namespaces, classes, dependency inversion (DIP), and dependency injection (DI). Adapt agreed decisions into the owning technical specs under **Technical spec** in `src/AGENTS.md`; keep each illustration below its governing rule. Only those project specs enter code metadata.

The fictitious flow announces an already recorded order and logs its event. Persistence and distributed delivery are outside this example.

## Example decisions

These obligations apply within the example; they are not framework defaults.

### Namespace boundaries

Each module **must** export one named namespace containing its public classes and interfaces. Namespace and class names **must** use PascalCase; methods and parameters **must** use camelCase. Imports **must** remain module-scoped; global namespace merging is prohibited.

### Class behavior

Application operations, event handlers, and adapters **must** be classes with instance methods and explicit return types. Event payloads **must** be readonly class properties. Free-standing function declarations are prohibited; inline callbacks **must** only forward arguments to methods. The composition entry point **must** be a static method.

### Local event delivery

Publishers **must** describe a fact with an event class and **must not** import subscribers. The local adapter **must** use Node.js `node:events` to dispatch `OrderPlaced` under `order.placed`, synchronously in subscription order. Subscribers **must not** return promises. With no subscribers, publication **must** do nothing; the first thrown handler error **must** propagate and stop remaining handlers. Delivery **must** remain in-process, without persistence or retries.

### Consumer-owned abstraction

The application **must** own `Orders.EventPublisher`; its adapter **must** implement that interface. The application **must not** import the adapter or `node:events`. The indirection's pain trigger is an anti-corruption boundary isolating the application's event contract from Node.js dispatch APIs.

### Constructor injection

The application **must** receive its publisher through the constructor. The concrete publisher, application operation, and subscriber **must** be instantiated and connected only in the composition entry point. Service locators and dependency injection containers are prohibited in this example.

## Illustration

Illustration of the decisions above, split by module. These paths describe an example project, not framework files to create automatically.

`src/orders/events.ts`

```typescript
export namespace OrderEvents {
  export class OrderPlaced {
    constructor(public readonly orderId: string) {}
  }
}
```

`src/orders/announce-order-placed.ts`

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

`src/messaging/local-events.ts`

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

`src/notifications/order-log.ts`

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

`src/main.ts`

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

Compile with TypeScript emission and Node.js types; namespaces require transformation and cannot use `erasableSyntaxOnly`. Expected output: `Order placed: order-1`.

The project toolchain spec owns compiler and runtime versions, configuration, and commands. If the project needs a broker or asynchronous delivery, decide transport, serialization, ordering, retries, idempotency, and transaction boundaries before implementing it.

Sources: [TypeScript namespaces](https://www.typescriptlang.org/docs/handbook/namespaces.html), [erasableSyntaxOnly](https://www.typescriptlang.org/tsconfig/erasableSyntaxOnly.html), [Node.js event delivery](https://nodejs.org/api/events.html#asynchronous-vs-synchronous).
