# Sources

The skill keeps the principle and drops the platform-specific format. Where a source mandates one language, that mandate is quoted and then not copied.

## Contract before code

- Government of Canada, "Contract First API Development Primer." Contract first starts from the consumer's requirements and then maps that contract onto the system. Code first starts from the implementation and generates the contract afterward. The contract is how consumers are to interact, so it is dictated by those consumers rather than by the shape of the backend. https://www.canada.ca/en/government/system/digital-government/digital-government-innovations/enabling-interoperability/api-guidance/contract.html
- Zalando RESTful API and Event Guidelines, "MUST follow API First principle." API First there means two things: define the API before coding the implementation, and get early feedback from peers and client developers. The API should separate what the caller needs from how the service is built, and stay stable even if the implementation is replaced. Zalando also requires OpenAPI as the specification language. This skill keeps the two things and does not keep the OpenAPI requirement. The carrier is whatever the current repo already uses. https://opensource.zalando.com/restful-api-guidelines/

## A caller can use it without the source

- OpenAPI Specification. A language-agnostic interface description lets humans and computers understand and call a service without its source code, extra docs, or captured traffic. An operation describes parameters, a request body, responses, and security. The responses cover a successful call and the known errors. https://spec.openapis.org/oas/latest.html
- Ian Robinson, "Consumer-Driven Contracts: A Service Evolution Pattern" (12 June 2006). Provider contracts are derived from what existing consumers actually rely on. The obligations come from outside the provider. The implementation stays free to change behind those obligations. https://martinfowler.com/articles/consumerDrivenContracts.html

## Protocol is not the contract

- Smithy 2.0. Smithy is protocol-agnostic: the model states how a client and a server understand an operation, and the transport and serialization are separate. An operation has input, output, and errors. Idempotency is a trait for operations a client may retry. Pagination is a trait for collections, with a continuation token the client follows until it is absent. AWS publishes its public service contracts as Smithy models, including operations, authentication, request and response types, and errors. https://smithy.io/2.0/spec/protocol-traits.html
- Smithy behavior traits: `@idempotent`, `@idempotencyToken`, and `@paginated`. https://smithy.io/2.0/spec/behavior-traits.html
- AWS, "Introducing AWS API models." Public AWS APIs are modeled before the SDK, and the model is the contract for operations and behaviors. https://aws.amazon.com/blogs/aws/introducing-aws-api-models-and-publicly-available-resources-for-aws-api-definitions/

This skill uses Smithy's split (semantics first, protocol second) and OpenAPI's test (a stranger can call it). It does not require either format.
