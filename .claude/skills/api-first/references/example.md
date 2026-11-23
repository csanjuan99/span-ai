# Worked example

A checkout client must place an order. The first sketch is an internal function that accepts a persisted row and returns it. The row includes `unit_price_cents`, `tax_row_id`, and a stack-bearing exception name. That is code first. The caller cannot use it without reading the function.

The contract below is what this skill returns before the handler exists. The carrier is boundary types because the sibling module already exports `PlaceOrderInput` and `PlaceOrderResult` and has no spec file.

```markdown
# Contract: PlaceOrder

**Status:** Ready
**Consumer:** Checkout client. Job: place one order for the cart the shopper already confirmed.
**Carrier:** types at `orders/place-order.contract.ts`

## Operation

PlaceOrder. The caller gets an order id and the status `placed`.

## Input

| Field | Required | Type and unit | Default |
| --- | --- | --- | --- |
| cartId | yes | opaque string | none |
| shipTo | yes | postal address | none |

## Output

| Field | Type and unit | Meaning for the caller |
| --- | --- | --- |
| orderId | opaque string | the id to show and to fetch later |
| status | `placed` | the order is stored once |

## Errors

| Code | When | Retry |
| --- | --- | --- |
| InvalidOrder | cartId or shipTo is missing or unusable | no |
| OrderAlreadyPlaced | this cart was already placed | no |

## Side effects

One order is stored for that cart. A second call with the same cart does not store another.

## Auth

The authenticated shopper who owns the cart. The caller presents the session the sibling checkout operations already use.

## Compatibility

- Callers may rely on: `orderId`, `status = placed`, and the two error codes.
- Breaking change: renaming those fields or codes, or storing two orders for one cart.

## Idempotency

Not applicable as a separate token. The cart id is the replay key. A repeat returns `OrderAlreadyPlaced` and does not write again.

## Pagination

Not applicable. The output is one order.

## Examples

### Success

Input: `{ cartId: "cart_1", shipTo: { country: "AR", line1: "Av. Corrientes 1" } }`
Output: `{ orderId: "ord_1", status: "placed" }`

### Error

Input: `{ cartId: "cart_1" }` with no shipTo.
Output code: `InvalidOrder`. Nothing is stored.

## Wire

Copied from `orders/place-order.handler.ts`: the sibling is a function `placeOrder(input)`.

## Boundary

Types live in `orders/place-order.contract.ts`. The handler maps them to the order store. `unit_price_cents` and `tax_row_id` stay inside the store. They are not on the output.

## Companion skills

none

## Checks

| Example | Check | Result |
| --- | --- | --- |
| Success | place-order stores one order and returns ord id | pending |
| Error | place-order rejects a missing address and stores nothing | pending |
```

What changed relative to the internal function: the caller sends a cart and an address, not a price row. The money calculation remains inside. The error is a code the client can branch on. The contract file is the source the handler must match.
