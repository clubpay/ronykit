# ex-12-ee: order service on rony + ee

An order/basket API served by `rony`, with state kept by `ee` (in-memory bus and
store; swap in `std/ee/store/postgres` and `std/ee/bus/nats` for production).

```sh
go run .   # http://127.0.0.1:8080, API docs at /docs
```

| Method | Path                            | Does                                                   |
|--------|---------------------------------|--------------------------------------------------------|
| GET    | `/items`                        | List items with available stock                        |
| PUT    | `/items/{id}/inventory`         | Back office: set units on hand                         |
| POST   | `/orders`                       | Open an order                                          |
| GET    | `/orders/{id}`                  | Get an order with its lines and total                  |
| POST   | `/orders/{id}/items`            | Add `quantity` of `itemId`; reserves stock             |
| DELETE | `/orders/{id}/items/{itemId}`   | Remove a line; releases its reservation                |
| POST   | `/orders/{id}/checkout`         | Submit; reserved stock is deducted, `submittedAt` set  |

See `shop.http` for runnable requests.

## Model

- `Item` (catalog), `Inventory` (per item: `OnHand`, `Reserved`), `Order`
  (`open` → `submitted`), `OrderLine` (one per item in an order).
- Available stock is `OnHand - Reserved`. Adding to an order reserves; the back
  office cannot set `OnHand` below what is reserved.

Each write is one ee unit of work on an aggregate, so it applies fully or not
at all:

- `BasketLine` (order + item inventory + line, loaded by `orderID:itemID`)
  backs add/remove. Concurrent reservations of the same item conflict on the
  inventory version and are retried against fresh stock, so it never
  oversells; the order root is always written, so a line change cannot slip
  past a concurrent checkout.
- `OrderAgg` (order + all lines + their inventories) backs checkout and reads.
