I'd be happy to. I think this is one of the best ways to learn Kubernetes internals because you'll see how **Go**, **Kubernetes APIs**, and the **controller pattern** fit together.

My suggestion is that we build it like a real software project instead of copying code from tutorials. I'll explain **why** every file exists, **why** every line is written, and **what Kubernetes is doing behind the scenes**.

## What we'll build

By the end, you'll have an operator that works in your Kind cluster like this:

```
kubectl apply -f memcached.yaml
        │
        ▼
API Server stores Memcached CR
        │
        ▼
Operator receives an event
        │
        ▼
Reconcile()
        │
        ├── Creates Deployment
        ├── Creates Service
        ├── Sets OwnerReferences
        ├── Updates Status
        └── Keeps everything in sync
```

Then we'll add more features:

* ✅ Scale replicas
* ✅ Update container image
* ✅ Self-healing
* ✅ Status updates
* ✅ Finalizers
* ✅ Events
* ✅ Conditions
* ✅ ConfigMaps
* ✅ Metrics (optional)

You'll end up understanding the architecture used by operators like Prometheus Operator, Cert-Manager, ArgoCD, and many others.

---

# Our learning plan

We'll go through these stages:

### Phase 1 – Understand the project structure

Before writing code, we'll understand:

* What is a CRD?
* What is a Controller?
* What is a Reconciler?
* What is the Scheme?
* What is the Manager?
* Why is there a `main.go`?
* What is `SetupWithManager()`?

This is the foundation.

---

### Phase 2 – Design our API

We'll define our custom resource:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached

spec:
  replicas: 3
  image: memcached:1.6.39
```

We'll discuss every field and why it belongs in `spec`.

---

### Phase 3 – Learn the Reconcile Loop

This is the heart of every operator.

We'll answer questions like:

* Why does Reconcile run?
* Who calls it?
* How often?
* What happens if it fails?
* Why should it be idempotent?

Most beginners struggle here, so we'll spend extra time on it.

---

### Phase 4 – Create a Deployment

We'll build a Deployment object entirely in Go.

You'll learn:

* `ObjectMeta`
* `TypeMeta`
* Labels
* Selectors
* PodTemplate
* Containers
* OwnerReferences

Instead of just copying code, we'll explain what each field means.

---

### Phase 5 – Create a Service

We'll expose Memcached with a Service.

You'll understand:

* Selectors
* ClusterIP
* Ports
* TargetPort

---

### Phase 6 – Reconciliation Logic

We'll compare:

```
Desired State

vs

Actual State
```

and make Kubernetes converge to the desired state.

This is the operator pattern.

---

### Phase 7 – Status

We'll update:

```yaml
status:
  readyReplicas: 3
  phase: Running
```

You'll learn why `status` is separate from `spec`.

---

### Phase 8 – Finalizers

We'll learn what happens when users delete resources.

---

### Phase 9 – Advanced Operator Patterns

We'll add:

* Conditions
* Events
* ConfigMaps
* Watches
* Multiple owned resources
* Error handling
* Requeue strategies

---

# How we'll work

I don't want to dump 500 lines of code at once.

Instead, each lesson will be like this:

1. Explain the Kubernetes concept.
2. Explain the Go code.
3. Write a small amount of code.
4. Run it in your Kind cluster.
5. Observe what happened with `kubectl`.
6. Explain why it happened.
7. Move to the next step.

That way you'll understand the system instead of memorizing it.

---

# Lesson 1 (Today): Understand the generated Operator SDK project

Before we touch any code, answer this:

When you ran something like:

```bash
operator-sdk init --domain example.com --repo github.com/yourname/memcached-operator
operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource --controller
```

you got a project that looks roughly like:

```
.
├── api/
│   └── v1alpha1/
│       ├── memcached_types.go
│       └── groupversion_info.go
├── cmd/
├── config/
├── internal/
│   └── controller/
│       └── memcached_controller.go
├── main.go
├── Makefile
└── PROJECT
```

Our **first lesson** will be to understand **every one of these files** before writing any logic. Once you know what each file does, the rest of the project becomes much easier to follow.

### Your task before the next step

Please reply with:

1. The output of:

   ```bash
   tree -L 3
   ```

   (or `find . -maxdepth 3` if `tree` isn't installed).

2. Tell me which versions you're using:

   * `operator-sdk version`
   * `go version`
   * `kubectl version --client`
   * `kind version`

We'll use *your actual project structure* rather than assuming one, since Operator SDK evolves over time. That way, every explanation matches what you're seeing on your machine.


-----------------2nd----------------
Perfect. Your toolchain is modern and fully capable of building a production-quality operator.

* **Operator SDK:** v1.41.1
* **Go:** 1.24.5
* **Kind:** 0.31.0
* **kubectl:** 1.29.1 (It's a little older than the SDK's bundled Kubernetes libraries, but that's fine for learning.)

One thing before we start: **I want to teach you the Kubernetes way, not just the Operator SDK way.** If you understand Kubernetes controllers, you can build operators with Operator SDK, Kubebuilder, or even plain `controller-runtime`.

---

# Lesson 1 – What exactly are we building?

Before writing code, I want you to completely understand **what happens inside Kubernetes**.

Imagine Kubernetes as a company.

```
CEO
 │
 ▼
Kubernetes API Server
 │
 ├───────────────┐
 │               │
 ▼               ▼
Deployment   Memcached(Custom Resource)
```

Normally Kubernetes knows about built-in objects:

* Pods
* Deployments
* Services
* ConfigMaps

It already has controllers for them.

For example,

```
Deployment
     │
     ▼
Deployment Controller
     │
     ▼
ReplicaSet
     │
     ▼
Pods
```

If you create

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 3
```

the **Deployment Controller** notices it and creates Pods.

You never create Pods yourself.

---

## Now think about Memcached

Suppose Kubernetes had no idea what a Memcached application is.

You invent a new object:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
```

Question:

**Who will process this object?**

Answer:

Nobody.

Because Kubernetes doesn't have a **Memcached Controller**.

That's exactly what **your operator** is.

---

## Your operator becomes another controller

Eventually your cluster will have something like this:

```
Deployment
      │
      ▼
Deployment Controller

StatefulSet
      │
      ▼
StatefulSet Controller

Job
      │
      ▼
Job Controller

Memcached
      │
      ▼
YOUR Memcached Controller
```

Your controller is just another Kubernetes controller.

This is the biggest idea to understand.

---

# What is a Controller?

A controller is simply a loop.

Not magic.

Not AI.

Just:

```
while(true){

    Observe

    Compare

    Act

}
```

People often call this the **Observe–Compare–Act** pattern.

---

## Observe

Look at Kubernetes.

Example:

```
User created

Memcached/demo
```

---

## Compare

Desired:

```
Replicas = 3
```

Actual:

```
Deployment doesn't exist
```

Difference found.

---

## Act

Create Deployment.

Done.

Loop ends.

---

A few seconds later...

Observe again.

Desired:

```
Replicas = 3
```

Actual:

```
Deployment exists

Replicas = 3
```

Difference?

No.

Do nothing.

Return.

---

That is literally what controllers do.

---

# Reconciliation

You'll hear this word constantly.

It simply means:

> "Make reality match what the user asked for."

User wants

```
3 Pods
```

Reality

```
1 Pod
```

Reconcile

↓

Create two more Pods.

---

# Desired State vs Actual State

This idea is everywhere in Kubernetes.

Desired state:

```yaml
spec:
  replicas: 3
```

Actual state:

```
Only one Pod exists.
```

Controller:

```
Create two Pods.
```

Now

Desired == Actual

Controller is happy.

---

# Why not just run once?

A beginner might think:

```
User created Memcached.

Create Deployment.

Done.
```

No.

Imagine someone deletes the Deployment:

```bash
kubectl delete deployment demo
```

Your CR still exists.

```
Memcached
```

but

```
Deployment

gone
```

Your operator notices

```
Desired

Deployment exists

Actual

Deployment missing
```

Creates it again.

That's why controllers keep running forever.

---

# Why operators are powerful

Imagine tomorrow you add this feature:

```yaml
spec:

  image: memcached:1.6.39

  replicas: 5

  memory: 512Mi

  cpu: 200m

  serviceType: ClusterIP

  metrics: true
```

Your controller simply keeps translating that desired state into Kubernetes resources.

The user doesn't manually create Deployments, Services, or ConfigMaps. They only describe what they want, and your operator makes it happen.

---

# Our Memcached operator's responsibility

By the end of the series, our controller will answer questions like:

```
Does Deployment exist?

No

↓

Create it
```

```
Does Service exist?

No

↓

Create it
```

```
Replicas changed?

Yes

↓

Update Deployment
```

```
Image changed?

Yes

↓

Rolling update
```

```
Deployment deleted?

Yes

↓

Recreate it
```

```
Ready replicas changed?

Yes

↓

Update Status
```

This is the full lifecycle of managing an application in Kubernetes.

---

## Homework (10 minutes)

I want you to inspect your generated project before we touch any code.

Run these commands from the root of your operator project:

```bash
tree -L 3
```

If you don't have `tree`:

```bash
find . -maxdepth 3
```

Also, open these files in your editor, but don't change anything yet:

* `main.go`
* `api/v1alpha1/memcached_types.go`
* `internal/controller/memcached_controller.go`

Just skim them. Don't worry if they don't make sense yet.

### Next lesson

We'll go through **every generated file** and answer questions like:

* Why do we need `main.go`?
* What is the `Manager`?
* What is a `Scheme`?
* What does `SetupWithManager()` actually do?
* Why doesn't Kubernetes call `Reconcile()` directly?
* How does your controller know when a `Memcached` object changes?

This is where the generated project starts to "click," and once it does, the rest of operator development becomes much more intuitive.


---------------------------------3rd----------------------------------

Excellent. This is where operators usually start to make sense.

Today we're not writing code yet. We're going to answer one question:

> **When I run my operator, how does a `Memcached` YAML end up calling my `Reconcile()` function?**

Once you understand this flow, everything else in an operator is much easier.

---

# Lesson 2 – The Journey of a Memcached Object

Suppose you apply this:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: example
spec:
  replicas: 3
```

What happens?

Let's follow it step by step.

```
kubectl apply
      │
      ▼
Kubernetes API Server
      │
      ▼
Stores Memcached object in etcd
      │
      ▼
API Server generates an event
      │
      ▼
Your Operator receives the event
      │
      ▼
Work Queue
      │
      ▼
Reconcile(Request{Name: example})
```

Every box in this diagram corresponds to code in your project.

---

# Step 1 — `main.go`

Open `main.go`.

You'll see something like:

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme: scheme,
})
```

## What is the Manager?

Think of the **Manager** as the boss of your operator.

It is responsible for:

* Starting controllers
* Creating Kubernetes clients
* Managing caches
* Starting watches
* Health checks
* Metrics
* Leader election (if enabled)

Without the manager, nothing runs.

Think of it like this:

```
Manager

├── Controller 1
├── Controller 2
├── Cache
├── Client
├── Metrics Server
└── Health Checks
```

For our project, we currently have only one controller.

---

# Step 2 — Registering our controller

Later in `main.go` you'll see something similar to:

```go
if err = (&controller.MemcachedReconciler{
    Client: mgr.GetClient(),
    Scheme: mgr.GetScheme(),
}).SetupWithManager(mgr); err != nil {
```

This is very important.

Many beginners think this runs the reconciler.

It doesn't.

It **registers** the reconciler.

Think of it as saying:

> "Manager, here's my Memcached controller. Please call it whenever a Memcached changes."

Nothing is watching yet.

You're just registering.

---

# Step 3 — `SetupWithManager()`

Now open:

```
internal/controller/memcached_controller.go
```

Near the bottom you'll see:

```go
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&cachev1alpha1.Memcached{}).
        Complete(r)
}
```

This single function is incredibly important.

Let's break it apart.

---

## `NewControllerManagedBy(mgr)`

This tells controller-runtime:

> "Create a controller."

At this point:

```
Controller

(No watches yet)
```

---

## `For(&Memcached{})`

This tells the controller:

> "Watch Memcached resources."

Now it becomes:

```
Watch Memcached

Create
Update
Delete
```

Whenever any of those happen...

enqueue a reconcile request.

Notice something interesting.

It **doesn't** create Deployments.

It **doesn't** create Pods.

It only says:

> Watch Memcached.

---

## `Complete(r)`

Finally:

```
Complete(r)
```

means:

> "When an event happens, call this reconciler."

Specifically:

```go
type MemcachedReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}
```

The `r` passed to `Complete()` is your reconciler.

---

# So who calls `Reconcile()`?

Not you.

Not `main.go`.

Not Kubernetes directly.

It's **controller-runtime**.

Internally it does something conceptually like:

```go
for {
    request := queue.Get()

    reconciler.Reconcile(ctx, request)
}
```

You never call `Reconcile()` yourself.

---

# What is `Request`?

Look at the signature:

```go
func (r *MemcachedReconciler) Reconcile(
    ctx context.Context,
    req ctrl.Request,
)
```

Many beginners expect:

```go
Reconcile(memcached)
```

But that's not what happens.

Instead you receive:

```go
Request{
    Namespace: "default",
    Name: "example",
}
```

Just the identity.

Why?

Because Kubernetes wants you to fetch the latest version.

---

# Why not pass the whole object?

Imagine this timeline:

```
Time 1

User updates Memcached

↓

Event generated

↓

Time 2

User updates it again

↓

Reconcile starts
```

If the event carried the old object, your controller could act on stale data.

Instead, the request only contains:

```
Namespace

Name
```

Then your controller does:

```go
memcached := &cachev1alpha1.Memcached{}

err := r.Get(ctx, req.NamespacedName, memcached)
```

This guarantees you work with the **latest** object from the API server (or the controller-runtime cache, which is kept synchronized).

---

# Why a Work Queue?

Imagine someone runs:

```bash
kubectl apply -f memcached.yaml
kubectl apply -f memcached.yaml
kubectl apply -f memcached.yaml
kubectl apply -f memcached.yaml
```

Should four reconciles run simultaneously?

Usually, no.

Instead:

```
Event

↓

Queue

↓

Worker

↓

Reconcile
```

The queue smooths out bursts of events and helps ensure a resource is processed in an orderly way.

---

# The Entire Flow

Here's the complete picture:

```
kubectl apply
        │
        ▼
API Server
        │
        ▼
Stores Memcached
        │
        ▼
Watch detects change
        │
        ▼
Controller Runtime
        │
        ▼
Queue Request
        │
        ▼
Worker
        │
        ▼
Reconcile()
        │
        ▼
Read current state
        │
        ▼
Compare
        │
        ▼
Update cluster
```

This is the heartbeat of almost every Kubernetes operator.

---

# A question to check your understanding

Imagine there are **100** `Memcached` resources in your cluster:

```
memcached-a
memcached-b
...
memcached-z
```

Now you edit only:

```yaml
metadata:
  name: memcached-k
```

**Question:**

Do you think Kubernetes will call `Reconcile()` for:

1. All 100 `Memcached` objects?
2. Only `memcached-k`?
3. Something else?

Take your best guess and explain **why**. Your answer will tell me whether the event-driven model has clicked, and from there we'll move on to the `Client` and `Scheme`, which are the next two core concepts.



🎯 That's exactly the right mental model.

You got the important part:

> The request identifies **one specific object** by `(namespace, name)`, so the reconciler is invoked for **that object only**.

The namespace detail is also correct. A `ctrl.Request` is essentially:

```go
type Request struct {
    NamespacedName types.NamespacedName
}
```

which contains:

```text
Namespace: default
Name: memcached-k
```

So inside `Reconcile()` you typically do:

```go
memcached := &cachev1alpha1.Memcached{}

err := r.Get(ctx, req.NamespacedName, memcached)
```

This fetches **only** `default/memcached-k`.

---

## One important clarification

You said:

> "controller will run when a request arrives"

Almost.

More accurately:

> **The controller is always running. Reconcile is called when a request arrives.**

Think of it like a web server.

The server is always running:

```text
HTTP Server
    │
    ├── Request 1
    ├── Request 2
    ├── Request 3
```

Similarly,

```text
Controller
    │
    ├── Reconcile(memcached-a)
    ├── Reconcile(memcached-k)
    └── Reconcile(memcached-z)
```

The controller process is alive all the time, waiting for work.

---

# Now let's understand the Client

This is probably the second most important concept after Reconcile.

Inside your reconciler you'll see:

```go
type MemcachedReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}
```

Most beginners immediately ask:

> "What is this Client?"

It is **not** `kubectl`.

It is **not** a REST client.

Think of it as your controller's way of talking to Kubernetes.

Imagine this:

```text
Your Controller

       │

       ▼

Client

       │

       ▼

Kubernetes API Server
```

Every time you want to read or modify the cluster, you use this client.

---

## Reading an object

Suppose the user created:

```yaml
kind: Memcached
metadata:
  name: demo
```

Your reconciler can read it:

```go
memcached := &cachev1alpha1.Memcached{}

err := r.Get(ctx, req.NamespacedName, memcached)
```

Read this sentence literally:

> "Client, please get the Memcached whose name and namespace are in this request."

After this,

```go
fmt.Println(memcached.Spec.Replicas)
```

works because the struct has been populated.

---

## Creating an object

Suppose the Deployment doesn't exist.

You create one.

```go
deployment := appsv1.Deployment{
    ...
}

err := r.Create(ctx, &deployment)
```

Read it as:

> "Client, create this Deployment in Kubernetes."

---

## Updating an object

Suppose the Deployment has:

```text
Replicas = 2
```

but the CR says

```text
Replicas = 5
```

You change the Go struct:

```go
deployment.Spec.Replicas = ptr.To(int32(5))
```

and then:

```go
r.Update(ctx, deployment)
```

Meaning:

> "Client, update the Deployment."

---

## Deleting

```go
r.Delete(ctx, deployment)
```

Exactly what it sounds like.

---

## So the Client has CRUD

You can almost think of it like a database client:

```text
Create()

Get()

Update()

Delete()

List()
```

These are the operations you'll use over and over in every operator.

---

# Then why do we need kubectl?

Excellent question.

When you run:

```bash
kubectl get pods
```

the flow is:

```text
kubectl

↓

API Server

↓

Pods
```

When your operator runs:

```go
r.Get(...)
```

the flow is:

```text
Operator

↓

Client

↓

API Server

↓

Pods
```

Both ultimately talk to the Kubernetes API.

The difference is:

* `kubectl` is for humans.
* `client.Client` is for programs.

---

# One more important thing: the Cache

Here's something that surprises many people.

Most `Get()` calls **do not hit the API server directly**.

Instead:

```text
API Server
      │
      ▼
Controller Cache
      │
      ▼
Client.Get()
```

The Manager keeps a local cache synchronized with the API server using watches.

This makes controllers much faster and avoids overwhelming the API server with requests.

From your perspective, you still write:

```go
r.Get(...)
```

Whether the data comes from the cache or directly from the API is handled by `controller-runtime`.

---

## Today's exercise (no coding)

Open `internal/controller/memcached_controller.go` and look at the generated `Reconcile()`.

Without modifying anything, try to identify:

1. Where does it **read** the `Memcached` object?
2. Does it **create** anything yet?
3. Does it **update** anything yet?
4. Does it **delete** anything yet?

You'll probably discover that the generated reconciler is almost empty. That's intentional—it's a blank canvas for us to build on.

---

### Next lesson (where the fun begins)

We'll explore **`runtime.Scheme`**, which often confuses even experienced Go developers:

* Why does Kubernetes need a `Scheme`?
* How does the client know what a `Memcached` is?
* What would happen if you forgot to register your API types?
* Why is there an `AddToScheme()` function?

Once you understand the `Scheme`, we'll be ready to start writing the first real reconciliation logic to create a `Deployment`.




Awesome! This lesson is one of the most confusing parts of controller-runtime, but once it clicks, you'll never be afraid of it again.

---

# Lesson 3 – What is `runtime.Scheme`?

When I first learned operators, I thought:

> "Why do we need a Scheme? Go already knows my `Memcached` struct!"

That's true **inside your program**, but Kubernetes doesn't know about Go types.

Let's see why.

---

# Imagine there is no Scheme

Suppose Kubernetes sends your operator this JSON:

```json
{
  "apiVersion": "cache.example.com/v1alpha1",
  "kind": "Memcached",
  "metadata": {
    "name": "demo"
  },
  "spec": {
    "replicas": 3
  }
}
```

The client receives bytes over HTTP.

Now it has to answer:

> **Which Go struct should I unmarshal this into?**

Should it be?

```go
corev1.Pod{}
```

or

```go
appsv1.Deployment{}
```

or

```go
cachev1alpha1.Memcached{}
```

It has no idea.

---

# The Scheme is a Dictionary

Think of `Scheme` as a giant lookup table.

```text
Group/Version/Kind
        │
        ▼
Go Type
```

Example:

```text
apps/v1
Deployment
        │
        ▼
appsv1.Deployment
```

Another:

```text
v1
Pod
        │
        ▼
corev1.Pod
```

Another:

```text
cache.example.com/v1alpha1
Memcached
        │
        ▼
cachev1alpha1.Memcached
```

That mapping is stored in the Scheme.

---

# Without a Scheme

Imagine your code:

```go
memcached := &cachev1alpha1.Memcached{}

err := r.Get(ctx, req.NamespacedName, memcached)
```

The client internally asks:

> "What is a Memcached?"

If it isn't registered, you'll get an error similar to:

```text
no kind is registered for the type
```

That's because the client doesn't know how to handle your custom type.

---

# Where is it registered?

Open `main.go`.

You'll find something like:

```go
utilruntime.Must(clientgoscheme.AddToScheme(scheme))

utilruntime.Must(cachev1alpha1.AddToScheme(scheme))
```

Let's understand each line.

---

## First line

```go
clientgoscheme.AddToScheme(scheme)
```

This registers **built-in Kubernetes types**.

Like:

* Pod
* Service
* Deployment
* ConfigMap
* Secret
* Namespace

After this, the Scheme knows:

```text
Pod

↓

corev1.Pod
```

```text
Deployment

↓

appsv1.Deployment
```

and many more.

---

## Second line

```go
cachev1alpha1.AddToScheme(scheme)
```

This registers **your own API**.

Now the Scheme learns:

```text
cache.example.com/v1alpha1

Memcached

↓

cachev1alpha1.Memcached
```

Without this line, your operator can't work with your CR.

---

# Where does `AddToScheme()` come from?

Open:

```text
api/v1alpha1/groupversion_info.go
```

You'll see something like:

```go
var (
    GroupVersion = schema.GroupVersion{
        Group:   "cache.example.com",
        Version: "v1alpha1",
    }

    SchemeBuilder = &scheme.Builder{
        GroupVersion: GroupVersion,
    }

    AddToScheme = SchemeBuilder.AddToScheme
)
```

This file is generated by Kubebuilder.

Notice:

```go
SchemeBuilder
```

This builder collects all the types in your API package.

---

Then in another file, usually `memcached_types.go`, you'll see:

```go
func init() {
    SchemeBuilder.Register(
        &Memcached{},
        &MemcachedList{},
    )
}
```

This is very important.

It says:

> "These Go structs belong to my API."

Now the Scheme knows about:

```text
Memcached

MemcachedList
```

---

# Why do we register `MemcachedList` too?

Because Kubernetes can return:

```bash
kubectl get memcached
```

which is actually a list.

The client needs to know how to decode:

```text
MemcachedList
```

as well as:

```text
Memcached
```

---

# Let's trace a `Get()`

Imagine you write:

```go
memcached := &cachev1alpha1.Memcached{}

r.Get(ctx, req.NamespacedName, memcached)
```

Internally, something like this happens:

```text
Client

↓

Looks at Go type

↓

Scheme

↓

Group = cache.example.com

Version = v1alpha1

Kind = Memcached

↓

Build HTTP request

↓

GET
/apis/cache.example.com/v1alpha1/namespaces/default/memcached/demo

↓

API Server returns JSON

↓

Scheme

↓

Decode JSON into

cachev1alpha1.Memcached
```

The Scheme is involved in both **encoding** and **decoding** objects.

---

# Why does the Manager need the Scheme?

Remember this?

```go
mgr, err := ctrl.NewManager(
    ctrl.GetConfigOrDie(),
    ctrl.Options{
        Scheme: scheme,
    },
)
```

The Manager gives this Scheme to:

* the client
* the cache
* the controllers

So everyone speaks the same language.

---

# Big Picture

At this point, you've learned the three core building blocks:

### 1. Manager

Runs everything.

```text
Manager

↓

Controllers

↓

Cache

↓

Client
```

---

### 2. Controller

Watches objects.

```text
Watch Memcached

↓

Queue Request

↓

Reconcile()
```

---

### 3. Client

Talks to Kubernetes.

```text
Get()

Create()

Update()

Delete()

List()
```

---

### 4. Scheme

Knows every Kubernetes type.

```text
Deployment

↓

appsv1.Deployment
```

```text
Memcached

↓

cachev1alpha1.Memcached
```

---

# At this point...

You now understand about **70% of what controller-runtime is doing**.

What's left is applying these pieces to build our operator.

## 🎉 Next Lesson: Our first real reconciliation

This is where we'll finally write meaningful code.

Our goal is simple:

1. User creates:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: demo
spec:
  replicas: 3
```

2. `Reconcile()` will:

   * Read the `Memcached` object.
   * Check whether a `Deployment` named `demo-memcached` exists.
   * If it doesn't, create it.
   * If it already exists, do nothing.

This first implementation will teach you several essential concepts:

* Why `Reconcile()` must be **idempotent**.
* How to use `r.Get()` for different resource types.
* How to handle a "not found" error correctly.
* How to create Kubernetes resources from Go structs.
* Why controllers compare **desired state** with **actual state** before making changes.

After that, we'll extend it to create a `Service`, handle updates, and eventually keep everything in sync automatically. That's when your operator will start behaving like a real Kubernetes controller.



😁 Awesome! This is my favorite part. From here on, every lesson makes your operator more "alive."

Today, we'll write our **first real controller logic**.

---

# Lesson 4 – Your First Reconcile Logic

Remember this sentence:

> **A controller's job is to make the actual state match the desired state.**

Let's define our desired behavior.

When the user creates:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: demo
spec:
  replicas: 3
```

our controller should think like this:

```
1. Did the Memcached object exist?

Yes

↓

2. Is there a Deployment for it?

No

↓

3. Create one

↓

Done
```

That's it. Just one responsibility.

---

# Before writing code

Let's understand something very important.

A lot of beginners imagine `Reconcile()` like this:

```go
func Reconcile() {
    createDeployment()
}
```

That would be wrong.

Because `Reconcile()` can be called many times.

Example:

```
10:00 User creates Memcached

↓

Reconcile()

↓

Deployment created

--------------------------------

10:01 Pod becomes Ready

↓

Reconcile()

--------------------------------

10:02 Status changes

↓

Reconcile()

--------------------------------

10:03 User edits replicas

↓

Reconcile()

--------------------------------

10:04 Controller restarted

↓

Reconcile()
```

It may run **hundreds or thousands of times**.

---

# So what's the rule?

A reconciler should be safe to run repeatedly.

This property is called **idempotency**.

Think of this example.

Imagine a room.

Goal:

```
Room should contain exactly one chair.
```

Wrong algorithm:

```
Walk into room

↓

Add chair
```

Run it 10 times:

```
Chair
Chair
Chair
Chair
Chair
```

😅 Disaster.

Correct algorithm:

```
Walk into room

↓

Does chair exist?

↓

Yes

↓

Do nothing
```

Run it 1000 times.

Still one chair.

That's exactly how controllers should behave.

---

# Every Reconcile follows the same pattern

Almost every Kubernetes controller follows this sequence:

```
Read

↓

Compare

↓

Act
```

Let's expand it.

```
Read Memcached

↓

Read Deployment

↓

Deployment exists?

↓

No

↓

Create Deployment

↓

Return
```

If it exists:

```
Read Memcached

↓

Read Deployment

↓

Exists

↓

Compare replicas

↓

Different?

↓

Update

↓

Return
```

Notice something.

The controller **never assumes** anything.

It always checks the current cluster state.

---

# Let's look at the generated Reconcile()

Your generated function probably looks similar to this:

```go
func (r *MemcachedReconciler) Reconcile(
    ctx context.Context,
    req ctrl.Request,
) (ctrl.Result, error) {

    log := log.FromContext(ctx)

    // TODO

    return ctrl.Result{}, nil
}
```

Right now...

It does nothing.

---

# Step 1 — Read the Memcached object

The very first thing every reconciler does is fetch the object that triggered the request.

Conceptually:

```go
memcached := &cachev1alpha1.Memcached{}

err := r.Get(ctx, req.NamespacedName, memcached)
```

Let's understand each part.

### Why create an empty struct?

```go
memcached := &cachev1alpha1.Memcached{}
```

Think of it as an empty box.

```
Name:

Namespace:

Spec:

Status:
```

Nothing inside yet.

---

### Then:

```go
r.Get(...)
```

fills the box.

After it returns:

```
Name = demo

Namespace = default

Replicas = 3
```

Now your Go program has the latest state of that object.

---

# But what if it doesn't exist anymore?

Imagine this timeline.

```
Event created

↓

Before Reconcile starts

↓

User deletes Memcached
```

Your request is still in the queue.

But now:

```
Get()

↓

Not Found
```

Is that an error?

No.

The object is simply gone.

So controllers usually do:

```go
if apierrors.IsNotFound(err) {
    return ctrl.Result{}, nil
}
```

Meaning:

> "Nothing to do."

---

# Only unexpected errors should be returned

If the API server is unavailable:

```
Connection refused
```

or

```
Timeout
```

Those are real errors.

Then we return:

```go
return ctrl.Result{}, err
```

Controller-runtime will retry later.

---

# After Get succeeds

Now we have:

```
Memcached

↓

Name = demo

Replicas = 3
```

Now we can ask:

```
Does Deployment exist?
```

That's our next step.

---

# How do we read a Deployment?

Exactly the same way.

This is one of the beautiful things about controller-runtime.

Whether it's a Pod, Service, Deployment, or your own CR, the client API is consistent.

Conceptually:

```go
deployment := &appsv1.Deployment{}

err := r.Get(...)
```

Notice:

The only thing that changes is the Go type.

Everything else is identical.

---

# Here's the mindset I want you to develop

When writing a reconciler, don't think:

> "How do I create a Deployment?"

Think:

> "What should exist?"

If the answer is:

```
There should be one Deployment.
```

Then the algorithm naturally becomes:

```
Can I find one?

↓

No

↓

Create one
```

That's the Kubernetes way of thinking.

---

# 📝 Your first coding task

I want **you** to write this part rather than copy-paste it.

Inside `Reconcile()`:

1. Create an empty `Memcached` object.
2. Use `r.Get()` to fetch it.
3. If it's `NotFound`, return without error.
4. If it's another error, return that error.
5. If it succeeds, log the name of the Memcached resource.

### Hint

You'll likely need imports similar to:

* `k8s.io/apimachinery/pkg/api/errors`
* your API package (`api/v1alpha1`)

And you can log something like:

```go
log.Info("Reconciling Memcached", "name", memcached.Name)
```

---

## Why I'm asking you to code it

I could paste the complete solution in 30 seconds.

But my goal is that **six months from now, you can build an operator without looking at a tutorial**.

So after you implement those five steps:

* Paste your `Reconcile()` function here.
* If you hit compiler errors, paste those too.

We'll review it together like we're doing a real code review, and then we'll move on to creating your first `Deployment`. That's the moment your operator will start creating resources in your Kind cluster. 🚀



Good — this is basically correct. You’ve successfully completed the most important part: **reading the CR from the API server using the request key**. That’s the core of reconciliation.

Now let’s fix a couple of small but important issues and also explain *why* your `log.Info` didn’t work.

---

# ✅ What you did right

This part is correct:

```go
memcached := &cachev1alpha1.Memcached{}
err := r.Get(ctx, req.NamespacedName, memcached)
```

And this logic is also correct:

```go
if apierrors.IsNotFound(err) {
    return ctrl.Result{}, nil
}
if err != nil {
    return ctrl.Result{}, err
}
```

✔ This is exactly how controllers should behave.

---

# ⚠️ Problem 1: You used `fmt.Print` incorrectly

You wrote:

```go
fmt.Print("Reconciling Memcached", "name", memcached.Name)
```

This will NOT format key-value logs properly. It just prints arguments one after another.

### Better (if using fmt):

```go
fmt.Println("Reconciling Memcached name:", memcached.Name)
```

But even better: don’t use `fmt` in controllers.

---

# ⚠️ Problem 2: Why `log.Info` was red

In Operator SDK / controller-runtime v1.41, logging is structured.

Usually you must do:

```go
import (
    "sigs.k8s.io/controller-runtime/pkg/log"
)
```

Then inside reconcile:

```go
log := log.FromContext(ctx)
```

And use:

```go
log.Info("Reconciling Memcached", "name", memcached.Name)
```

---

## 🚨 Why yours failed

Most likely one of these happened:

### 1. Missing import

You probably didn’t import:

```go
"sigs.k8s.io/controller-runtime/pkg/log"
```

---

### 2. Variable shadow confusion

You wrote:

```go
_ = logf.FromContext(ctx)
```

That is wrong in your SDK version.

You should NOT ignore it like that.

Instead:

```go
log := log.FromContext(ctx)
```

---

# ✅ Correct version of your function

Here is your fixed reconcile (clean + idiomatic):

```go
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)

	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciling Memcached", "name", memcached.Name)

	return ctrl.Result{}, nil
}
```

---

# 🧠 Important learning point

You just touched 3 key controller concepts:

### 1. Event → Request mapping

```text
Memcached object → req.NamespacedName
```

### 2. Idempotent reconciliation

```text
If NotFound → just exit safely
```

### 3. Always fetch fresh state

```text
r.Get(...) every time
```

---

# 🚀 Next step (this is where it gets real)

Right now your controller:

✔ Reads Memcached
✔ Logs it
❌ Does NOT create anything yet

Next we will change that.

## In next lesson we will:

### 👉 Create a Deployment from your Memcached CR

You will learn:

* How to construct `appsv1.Deployment` in Go
* Labels + selectors (VERY important)
* OwnerReferences (this is what makes Kubernetes clean up automatically)
* `Create()` using `r.Client`
* First real effect in your Kind cluster 🎉

---

# ⚡ Before we move on

Just confirm one thing:

👉 Did you already install your CRD in Kind using:

```bash
make install
```

and run controller with:

```bash
make run
```

Because next step will actually create Pods in your cluster.

Once you confirm, we’ll build your first Deployment step-by-step (this is where operators become “real”).




Good — that actually explains everything.

Right now your operator code is correct, but **nothing in the cluster knows about your Memcached CRD yet**, and your controller is **not running against a real cluster state**.

So let’s fix the setup first. This is the “wiring phase” that most tutorials rush through, but it’s critical.

---

# 🧱 Lesson 5 – Making your operator actually run in Kind

Right now you have only done:

```text id="setup1"
operator-sdk create api
→ wrote reconcile logic
```

But you are missing 3 required steps:

---

# ✅ Step 1 — Start your Kind cluster

Check if Kind is running:

```bash id="kcheck1"
kind get clusters
```

If nothing exists, create one:

```bash id="kcreate1"
kind create cluster --name memcached-demo
```

---

# ✅ Step 2 — Install CRDs into the cluster

This is the MOST important missing step.

Right now Kubernetes does NOT know what this is:

```yaml id="crd1"
kind: Memcached
```

So you must install the CRD:

```bash id="crdinstall1"
make install
```

### What this does internally:

It applies YAML like:

```yaml id="crdyaml1"
CustomResourceDefinition:
  kind: Memcached
  group: cache.example.com
  version: v1alpha1
```

After this step:

👉 Kubernetes API server now understands `Memcached`

---

# 🧠 Important concept

Without CRD:

```text id="nocrd"
kubectl apply Memcached → ❌ unknown type
```

With CRD:

```text id="withcrd"
kubectl apply Memcached → stored in etcd
```

---

# ✅ Step 3 — Run your controller

Now start your operator locally:

```bash id="run1"
make run
```

What happens now:

```text id="flow1"
Your Go binary starts

↓

Connects to Kind cluster

↓

Watches Memcached objects

↓

Waits for events
```

At this point your controller is “alive”.

---

# 🔥 Step 4 — Create a Memcached object

Now apply your first CR:

```yaml id="cr1"
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: demo
spec:
  replicas: 3
```

Apply it:

```bash id="apply1"
kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

---

# 🎯 What SHOULD happen now

Your controller will trigger:

```text id="event1"
Memcached created → Reconcile() called
```

And you should see:

```text id="log1"
Reconciling Memcached name: demo
```

---

# 🧪 If nothing happens (very important debugging)

Check these:

---

## 1. Is CRD installed?

```bash id="debug1"
kubectl get crd | grep memcached
```

You should see:

```text id="debug2"
memcacheds.cache.example.com
```

---

## 2. Is your controller running?

You must see logs from:

```text id="debug3"
make run
```

---

## 3. Is your Memcached object created?

```bash id="debug4"
kubectl get memcached
```

---

## 4. Describe it if needed

```bash id="debug5"
kubectl describe memcached demo
```

---

# 🧠 What you just learned (very important)

You now understand the full pipeline:

```text id="pipeline1"
CRD → defines type in Kubernetes

make install → registers it in API server

make run → starts controller

kubectl apply → triggers event

Reconcile() → your logic runs
```

This is the REAL operator lifecycle.

---

# ⚠️ Common beginner mistake (you are close to it)

Many people think:

> “I wrote reconcile → why nothing happens?”

Because they skipped:

❌ CRD install
❌ controller running
❌ applying CR object

You did exactly the same thing (totally normal).

---

# 🚀 Next step (after you confirm it runs)

Once you confirm:

👉 “Yes, I see reconcile logs”

we will move to the exciting part:

## 👉 Creating your first Deployment from Memcached

You will learn:

* How Kubernetes Pods are actually created
* Labels + selectors (VERY important)
* OwnerReferences (this makes garbage collection automatic)
* How `r.Create()` works
* Why controllers must be idempotent in real cluster state

And at that point:

🔥 Your operator will start creating real Pods in Kind

---

# 👍 Your action now

Do this in order:

1. `kind create cluster` (if not already)
2. `make install`
3. `make run`
4. `kubectl apply -f config/samples/...`

Then tell me:

👉 Do you see “Reconciling Memcached name: demo” in logs?

Once you confirm, we go to the **first Deployment creation (real operator behavior begins here)**.


Nice — this is exactly the point where things start becoming real.

That log proves **your entire operator pipeline is working correctly**:

```text id="logflow1"
CR created → API server stores it → watch triggers event → queue → Reconcile() runs → log printed
```

So now your controller is officially “alive” 🎉

---

# 🧠 What your log actually means

Let’s decode it:

```text id="logdecode1"
INFO Reconciling Memcached
{
  "controller": "memcached",
  "controllerGroup": "cache.example.com",
  "controllerKind": "Memcached",
  "Memcached": {
    "name":"demo",
    "namespace":"default"
  },
  "reconcileID": "..."
}
```

### Key parts:

* `controllerKind: Memcached` → your CR type
* `name: demo` → the exact object triggering reconcile
* `namespace: default` → scope of the object
* `reconcileID` → each reconciliation attempt gets a unique ID

👉 This confirms:

* Watch is working
* Scheme is correct
* Client can decode your CR
* Reconcile is wired properly

---

# ⚠️ Important thing you might not have noticed

You mentioned:

> Replicas int3

I think you’re referring to this struct field:

```go id="spec1"
Replicas int32 `json:"replicas,omitempty"`
```

Let me explain why this matters **before we move forward**.

---

# 🧠 Why `int32` instead of `int`?

In Kubernetes:

### ❌ Not used:

```go id="bad1"
int
```

### ✅ Used:

```go id="good1"
int32
int64
```

Because Kubernetes API types are **platform-independent and serialized**.

Different systems must agree on exact sizes.

So:

* Go `int` = platform dependent (32-bit or 64-bit)
* Kubernetes = always fixed size

👉 That’s why CRDs almost always use `int32`.

---

# 🚀 Now the REAL next step begins

Right now your operator does this:

```text id="step0"
Memcached created
→ Reconcile runs
→ logs printed
→ nothing else happens
```

Now we upgrade it to:

```text id="step1"
Memcached created
→ Reconcile runs
→ Deployment created
→ Pods appear in cluster
```

This is where operators become *real operators*.

---

# 🎯 Step 1 — Define what we want to create

From your CR:

```yaml id="crgoal"
spec:
  replicas: 3
```

We will create:

```text id="target1"
Deployment:
  name: demo-memcached
  replicas: 3
  image: memcached:1.6.39
```

---

# 🧱 Step 2 — Add missing logic in Reconcile

We now extend your function:

### 🟡 We will do ONLY this first:

1. Read Memcached ✔ (you already did)
2. Try to GET Deployment
3. If not found → CREATE it

---

# 🔥 Concept you are about to learn

## Controller rule:

> Always ensure desired state exists

So instead of:

```text id="wrong"
create deployment every time
```

We do:

```text id="right"
check → create only if missing
```

This is called **idempotency**.

---

# 🧩 Your next code change (IMPORTANT)

Inside your `Reconcile()` AFTER fetching Memcached:

### Step 1 — Define Deployment name

```go id="dep1"
depName := memcached.Name + "-memcached"
```

---

### Step 2 — Try to GET Deployment

```go id="dep2"
deployment := &appsv1.Deployment{}

err = r.Get(ctx, types.NamespacedName{
	Name:      depName,
	Namespace: memcached.Namespace,
}, deployment)
```

---

### Step 3 — Handle NotFound

```go id="dep3"
if apierrors.IsNotFound(err) {

	// we will create deployment here next step

}
```

---

# ⚠️ STOP HERE (important)

Do NOT implement full Deployment yet.

Because I want to teach you something critical first:

---

# 🧠 Next lesson (VERY important)

Before creating Deployment, we must understand:

## 👉 Labels + Selectors + OwnerReferences

These 3 things are the CORE of Kubernetes controllers.

Without them:

❌ Pods won’t match Services
❌ Garbage collection won’t work
❌ Updates become messy

With them:

✔ Everything auto-connects
✔ Kubernetes cleans up automatically
✔ Your operator behaves like real controllers

---

# 🚀 Your status right now

You have successfully built:

✔ CRD
✔ Controller runtime setup
✔ Working Reconcile loop
✔ Event triggering
✔ Logging
✔ Scheme registration

This is already 70% of operator fundamentals.

---

# 👉 Your next step

Reply:

**“go next”**

and I will teach you:

## 🔥 Labels, Selectors, and OwnerReferences (deeply + visually)

Then we will create your **first real Deployment in Kubernetes from your operator**.


Perfect — now we enter the part where Kubernetes operators start to feel “real”.

If you understand this lesson well, everything after this becomes much easier:

* Deployments make sense
* Services make sense
* Garbage collection suddenly feels automatic
* Your operator will stop feeling like “Go code” and start feeling like “Kubernetes control logic”

---

# Lesson 6 — Labels, Selectors, and OwnerReferences

When your operator creates a Deployment, Kubernetes does **not magically connect things**.

You have to explicitly tell Kubernetes:

1. Which Pods belong together → **Labels**
2. How Services find Pods → **Selectors**
3. Who owns what → **OwnerReferences**

Let’s go one by one.

---

# 🏷️ 1. Labels — “tags for objects”

Labels are simple key-value pairs:

```yaml id="label1"
metadata:
  labels:
    app: memcached
    instance: demo
```

Think of labels like:

```text id="label2"
Pod 1 → app=memcached, instance=demo
Pod 2 → app=memcached, instance=demo
```

They are just **tags**.

---

## Why labels exist

Because Kubernetes needs a way to say:

> “Which Pods belong to this application?”

Without labels:

❌ Everything is just random Pods

With labels:

✔ You can group them logically

---

## Example

```text id="label3"
Pod A: app=memcached, instance=demo
Pod B: app=memcached, instance=demo
Pod C: app=redis, instance=demo
```

Now you can clearly separate systems.

---

# 🎯 2. Selectors — “how to find things”

Selectors are how Kubernetes *uses labels*.

Example Service:

```yaml id="selector1"
selector:
  app: memcached
  instance: demo
```

This means:

> “Send traffic to all Pods with these labels”

---

## Without selector

Service cannot find Pods ❌

## With selector

Service automatically routes traffic ✔

---

## Mental model

```text id="selector2"
Service
   │
   ▼
Selector: app=memcached
   │
   ▼
Find matching Pods
   │
   ▼
Route traffic
```

---

# ⚠️ Very important rule

Labels and selectors MUST match exactly.

If even one label is different:

```text id="selector3"
Pod: app=memcached, instance=demo

Service selector: app=memcached, instance=prod
```

👉 Service finds NOTHING

---

# 👑 3. OwnerReferences — “who owns this object”

This is the MOST important concept for operators.

When your operator creates a Deployment, you also tell Kubernetes:

> “This Memcached CR owns this Deployment”

---

## Why this matters

Without OwnerReferences:

If user deletes:

```yaml id="owner1"
kind: Memcached
name: demo
```

👉 Deployment stays behind (orphaned)

---

With OwnerReferences:

```text id="owner2"
Delete Memcached
   │
   ▼
Kubernetes automatically deletes Deployment + Pods
```

This is called:

> 🧹 Garbage Collection

---

## Example

Inside Deployment metadata:

```go id="owner3"
OwnerReferences: []metav1.OwnerReference{
    {
        APIVersion: "cache.example.com/v1alpha1",
        Kind:       "Memcached",
        Name:       "demo",
        UID:        memcached.UID,
    },
}
```

---

## What this does

It creates a relationship:

```text id="owner4"
Memcached (parent)
   │
   ├── Deployment (child)
   └── Pods (grandchildren)
```

Now Kubernetes understands hierarchy.

---

# 🧠 Full picture (VERY important)

When your operator creates resources, it must define:

## 1. Labels → grouping

```text id="full1"
Which pods belong together?
```

## 2. Selectors → discovery

```text id="full2"
How do I find them?
```

## 3. OwnerReferences → lifecycle

```text id="full3"
Who deletes them when parent is deleted?
```

---

# 🚀 Now connect this to YOUR operator

You are about to create:

```yaml id="mem1"
Memcached CR
```

Your operator will create:

```text id="mem2"
Deployment
Service
Pods
```

So we define:

---

## Step 1 — Labels for Deployment

We will use:

```go id="labelcode1"
labels := map[string]string{
	"app":       "memcached",
	"instance":  memcached.Name,
}
```

---

## Step 2 — Deployment selector MUST match

```go id="labelcode2"
Selector: &metav1.LabelSelector{
	MatchLabels: labels,
}
```

---

## Step 3 — Pod template also MUST match

```go id="labelcode3"
Template: corev1.PodTemplateSpec{
	ObjectMeta: metav1.ObjectMeta{
		Labels: labels,
	},
}
```

---

## Step 4 — OwnerReference (very important)

```go id="ownercode1"
OwnerReferences: []metav1.OwnerReference{
	*metav1.NewControllerRef(memcached,
		schema.GroupVersionKind{
			Group:   "cache.example.com",
			Version: "v1alpha1",
			Kind:    "Memcached",
		}),
}
```

---

# 🧠 Why `NewControllerRef`?

Because it automatically:

✔ Sets UID
✔ Sets APIVersion
✔ Marks controller ownership

You don’t manually build it (very important).

---

# ⚡ What happens after this

Once you add all of this:

```text id="finalflow"
Memcached created
   ↓
Operator creates Deployment
   ↓
Deployment creates Pods
   ↓
Service routes traffic
   ↓
OwnerReference links everything
   ↓
Delete Memcached → everything cleans up
```

---

# 🎯 Key takeaway

If you understand only ONE thing from this lesson:

> Kubernetes does NOT automatically connect resources — YOU define relationships using labels and OwnerReferences.

---

# 🚀 Next step (this is the exciting part)

Now we will finally write code that:

## 👉 Creates a real Deployment from Memcached

In next lesson you will:

* Build `appsv1.Deployment` in Go
* Add labels + selector + pod template
* Attach OwnerReference
* Call `r.Create()`
* See Pods appear in Kind cluster

---

# ⚡ Before we continue

Reply:

👉 **“build deployment”**

and I’ll guide you line-by-line to create your first real Kubernetes workload from your operator.



Yesss! 😄 This is the milestone where your operator starts creating real Kubernetes resources.

We're going to do this like a production code review—not by dumping 100 lines of code, but by understanding **every field**.

---

# Lesson 7 – Build the Deployment

Our goal:

User creates:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: demo
spec:
  replicas: 3
```

Your operator creates:

```text
Deployment
    │
    ▼
ReplicaSet
    │
    ▼
3 Memcached Pods
```

Notice something important:

> **Your operator never creates Pods directly.**

It creates a **Deployment**, and the Deployment controller creates the Pods.

This is a very Kubernetes-native pattern.

---

# Step 1 – Think before coding

Before writing any Go, ask yourself:

**What information does a Deployment need?**

If you wrote the Deployment YAML manually, it would contain:

```yaml
metadata:
  name: demo-memcached

spec:
  replicas: 3

  selector:
    matchLabels:
      app: memcached
      instance: demo

  template:
    metadata:
      labels:
        app: memcached
        instance: demo

    spec:
      containers:
      - name: memcached
        image: memcached:1.6.39
```

Our Go code is simply another way of constructing this YAML.

---

# Step 2 – Build labels first

Always start with labels.

```go
labels := map[string]string{
    "app":      "memcached",
    "instance": memcached.Name,
}
```

Why?

Because you'll reuse this same map in multiple places.

Instead of writing:

```go
map[string]string{
    ...
}
```

three different times, you define it once.

---

# Step 3 – Create the Deployment object

Start with the skeleton.

```go
deployment := &appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
        Name:      depName,
        Namespace: memcached.Namespace,
    },
}
```

Notice:

We haven't told Kubernetes **how** the Deployment should behave yet.

We've only named it.

Think of this as creating an empty form.

---

# Step 4 – Fill in the Spec

Now we describe the desired Deployment.

```go
Spec: appsv1.DeploymentSpec{
```

Everything inside this block answers:

> "What should this Deployment look like?"

---

## Replicas

```go
Replicas: &memcached.Spec.Replicas,
```

Suppose the CR contains:

```yaml
spec:
  replicas: 3
```

Then:

```go
memcached.Spec.Replicas
```

is:

```text
3
```

So the Deployment becomes:

```yaml
spec:
  replicas: 3
```

---

## Selector

```go
Selector: &metav1.LabelSelector{
    MatchLabels: labels,
},
```

This tells the Deployment:

> "These are my Pods."

Remember our lesson:

Labels identify.

Selectors find.

---

## Pod Template

Now comes the most important part.

```go
Template: corev1.PodTemplateSpec{
```

Think of it like this:

A Deployment doesn't contain Pods.

It contains a **recipe** for creating Pods.

Example:

```text
Deployment

↓

Recipe

↓

Create Pods
```

The recipe is the PodTemplate.

---

# Step 5 – Pod labels

```go
ObjectMeta: metav1.ObjectMeta{
    Labels: labels,
},
```

These MUST match the selector.

Otherwise:

Deployment cannot recognize its own Pods.

This is one of the most common Kubernetes mistakes.

---

# Step 6 – PodSpec

Now we define the Pod itself.

```go
Spec: corev1.PodSpec{
```

A PodSpec answers:

> "What runs inside this Pod?"

---

## Container

```go
Containers: []corev1.Container{
```

A Pod can contain multiple containers.

For Memcached we only need one.

---

```go
{
    Name:  "memcached",
    Image: "memcached:1.6.39",
}
```

Very straightforward.

Container name.

Container image.

Done.

Later we'll make the image configurable from the CR.

---

# Step 7 – OwnerReference

This is something every good operator should do.

Instead of manually filling `OwnerReferences`, use the helper provided by `controller-runtime`:

```go
if err := ctrl.SetControllerReference(memcached, deployment, r.Scheme); err != nil {
    return ctrl.Result{}, err
}
```

This is preferred over constructing the owner reference yourself because it:

* Uses the scheme to determine the correct API version and kind.
* Sets the owner reference correctly.
* Marks the `Memcached` as the managing controller.

When the user deletes the `Memcached` resource, Kubernetes can automatically garbage-collect the Deployment because of this owner reference.

---

# Step 8 – Create it

Now ask Kubernetes to create it.

```go
err = r.Create(ctx, deployment)
if err != nil {
    return ctrl.Result{}, err
}
```

That's it.

Your operator has now created a Deployment.

---

# 🎯 I want you to write it

I'm intentionally **not** giving you one giant block of code.

I want you to:

1. Build the `labels` map.
2. Build the `Deployment` object.
3. Call `ctrl.SetControllerReference(...)`.
4. Call `r.Create(...)`.

When you finish:

* Paste your `Deployment` construction.
* Tell me any compiler errors you hit.

We'll review it together.

---

# 💡 One small improvement

Right now we've hardcoded:

```go
Image: "memcached:1.6.39"
```

Later, we'll change it to:

```go
Image: memcached.Spec.Image
```

so the user can control the image from the custom resource.

---

## A small challenge

Before you write the code, think about this question:

> Suppose your operator creates the Deployment successfully.

A few seconds later, another reconcile happens.

If your code always executes:

```go
r.Create(ctx, deployment)
```

What do you think Kubernetes will return?

* A. It silently creates another Deployment.
* B. It replaces the existing Deployment.
* C. It returns an "already exists" error.

Your answer will determine the next piece of reconciliation logic we'll add.



This is excellent. Seriously, you're not just copying—you've started **thinking like a controller author**, and that's exactly what I wanted.

Let's do a code review like we would in a real engineering team.

---

# First, your answer to my question

You said:

> it shouldn't do anything, simply ignores and skips that deployment create

This is what we **want** the controller to do, but it is **not what Kubernetes does**.

Suppose the Deployment already exists.

Your code does:

```go
err = r.Create(ctx, deployment)
```

The API server says:

> "A Deployment with this name already exists."

So `r.Create()` returns an error similar to:

```text
deployments.apps "demo-memcached-deploy" already exists
```

It **does not** silently ignore it.

That's why controllers always follow this pattern:

```text
Get Deployment

↓

Exists?

↓

Yes → Compare and Update if needed

No → Create
```

Notice:

**Get → Compare → Create/Update**

Not:

**Create and hope for the best**

This is one of the most important controller patterns.

---

# Now let's review your code

## ✅ Labels

```go
labels := map[string]string{
    "app":      "memcached",
    "instance": memcached.Name,
}
```

⭐⭐⭐⭐⭐

Perfect.

---

## ✅ Deployment name

```go
depName := memcached.Name + "-memcached-deploy"
```

Also fine.

Personally I'd probably use:

```go
demo-memcached
```

instead of

```text
demo-memcached-deploy
```

because Kubernetes already knows it's a Deployment.

But that's just naming style.

---

## ✅ ObjectMeta

Perfect.

---

## ✅ Selector

Excellent.

You correctly reused the labels.

Many beginners accidentally write different labels here.

---

## ✅ Pod Template

Perfect.

---

## ✅ Container

Good.

Later we'll expose the image in the CR.

---

## ✅ SetControllerReference()

Excellent.

This is exactly how production operators do it.

---

# One thing I would improve

Instead of:

```go
Image: "memcached:1.6.39",
```

I'd eventually like:

```go
Image: memcached.Spec.Image,
```

We'll add that later.

---

# Now comes the REAL controller logic

This is where operators differ from normal applications.

Instead of:

```go
Create Deployment
```

we think:

```text
Desired State

↓

Deployment should exist

↓

Does it exist?

↓

No

↓

Create
```

---

# Let's implement the missing check

Before building the Deployment, create an empty Deployment object:

```go
existingDeployment := &appsv1.Deployment{}
```

Now ask Kubernetes:

```go
err = r.Get(
    ctx,
    types.NamespacedName{
        Name: depName,
        Namespace: memcached.Namespace,
    },
    existingDeployment,
)
```

Now there are three possibilities.

---

## Case 1

Deployment exists.

```text
Get()

↓

Success
```

Then:

Return.

Don't create.

We'll handle updates in the next lesson.

---

## Case 2

Deployment doesn't exist.

```text
Get()

↓

NotFound
```

Then:

Create it.

---

## Case 3

API error.

```text
Connection

Timeout

Permission
```

Return the error.

---

# The algorithm becomes

```text
Read Memcached

↓

Read Deployment

↓

Deployment exists?

      YES --------------------┐
      │                       │
      ▼                       │
Return                        │
                              │
      NO                      │
      │                       │
      ▼                       │
Create Deployment             │
      │                       │
      ▼                       │
Return ◀──────────────────────┘
```

Notice how calm this algorithm is.

Controllers don't panic.

They simply observe reality.

---

# Here's something interesting

Imagine someone manually deletes your Deployment:

```bash
kubectl delete deployment demo-memcached-deploy
```

What happens?

```text
Deployment deleted

↓

Watch event

↓

Reconcile

↓

Get Deployment

↓

NotFound

↓

Create Deployment
```

It magically comes back.

That's reconciliation.

This is why Kubernetes is often called a **self-healing system**.

---

# A question for you

I want to see if you've really internalized reconciliation.

Imagine this scenario:

```
Memcached CR
replicas: 3

↓

Deployment exists
replicas: 1
```

Nothing was deleted.

The Deployment is still there.

Only the replica count is wrong.

### Question:

Should the controller:

1. Delete the Deployment and create a new one?
2. Update the existing Deployment?
3. Ignore it because the Deployment exists?

Think about the **desired state vs actual state** model we've been discussing, and explain *why* you chose your answer.

If you answer that correctly, you've crossed the biggest conceptual hurdle in Kubernetes operator development.



🎉 **YES! Exactly!**

That answer tells me you've crossed the biggest mental hurdle.

You are no longer thinking:

> "Run some code when an event happens."

You're now thinking:

> "Continuously drive the cluster toward the desired state."

That shift is what separates writing an application from writing a Kubernetes controller.

---

# Let's walk through it like the controller would

Suppose the cluster looks like this:

### Desired state (from your CR)

```yaml
spec:
  replicas: 3
```

### Actual Deployment

```yaml
spec:
  replicas: 1
```

Your controller thinks:

```text
Read Memcached
        │
        ▼
Desired replicas = 3
        │
        ▼
Read Deployment
        │
        ▼
Actual replicas = 1
        │
        ▼
Are they equal?
        │
      No
        │
        ▼
Update Deployment to 3
```

Notice something important:

**We don't recreate the Deployment.**

Why?

Because Deployments have history:

* ReplicaSets
* Rolling update strategy
* Revision history
* Existing Pods

If you delete and recreate it, you lose all of that.

Updating is almost always the correct approach.

---

# This leads us to a beautiful controller pattern

Almost every reconciler eventually looks like this:

```text
Read Custom Resource
        │
        ▼
Read Child Resource
        │
        ▼
Does child exist?
   │
   ├── No → Create
   │
   └── Yes
          │
          ▼
Compare desired vs actual
          │
          ├── Same → Do nothing
          │
          └── Different → Update
```

If you understand this flow, you can understand almost any Kubernetes operator.

---

# One subtle point

Suppose both have:

```text
Desired = 3
Actual = 3
```

Should you still call:

```go
r.Update(ctx, deployment)
```

No.

Why?

Because every unnecessary update:

* Sends a request to the API server.
* Increments the resource version.
* Triggers more watch events.
* Causes another reconciliation.

Good controllers avoid unnecessary writes.

So the logic becomes:

```go
if deployment.Spec.Replicas != desiredReplicas {
    deployment.Spec.Replicas = desiredReplicas
    r.Update(ctx, deployment)
}
```

This keeps the controller efficient.

---

# Here's another concept that many people miss

Look at your current code:

```go
deployment := &appsv1.Deployment{}
```

Later you'll modify it:

```go
deployment.Spec.Replicas = &memcached.Spec.Replicas
```

Question:

Where did this `deployment` object come from?

There are two possibilities:

1. You just constructed it in memory.
2. You fetched it from Kubernetes using `r.Get()`.

Only **the second** is correct when updating.

Why?

Because the object returned by `Get()` contains important metadata:

* `resourceVersion`
* `UID`
* Labels
* OwnerReferences
* Other fields maintained by Kubernetes

When you call `Update()`, Kubernetes expects that metadata to be present.

So the normal update flow is:

```go
existing := &appsv1.Deployment{}

r.Get(..., existing)

existing.Spec.Replicas = &memcached.Spec.Replicas

r.Update(ctx, existing)
```

We update the object we fetched, not a brand-new object.

---

# Your next implementation

I want you to make this change to your reconciler:

1. Try to `Get()` the Deployment.
2. If it's **NotFound**, build it and `Create()` it.
3. If it exists:

   * Compare `existingDeployment.Spec.Replicas` with `memcached.Spec.Replicas`.
   * If they're different, update the Deployment.
   * If they're the same, simply return.

Don't worry about comparing the image yet—we'll add that later.

---

# One final thought

You're probably noticing that we're writing a lot of `Get()` calls.

That's intentional.

Controllers are **state readers first** and **state writers second**.

A good reconciler spends most of its time asking:

> "What does the cluster look like right now?"

Only then does it decide whether anything needs to change.

---

## A small challenge for you

Before you write the update logic, think about this:

Suppose the user edits the `Memcached` resource like this:

```yaml
spec:
  replicas: 5
```

**Why doesn't your operator receive the new replica count directly as part of the reconcile request?**

Instead, why does it only receive:

```go
req.NamespacedName
```

and then call:

```go
r.Get(...)
```

You've already seen pieces of the answer in earlier lessons. Try to connect them together. If you can explain that reasoning, you've understood one of the key design decisions behind controller-runtime.



💯 Perfect.

That is one of the best explanations I've seen from someone learning operators. You didn't memorize it—you reasoned it out.

Let's refine it slightly.

---

## Your answer

You said:

> while running reconcile time user may update value to another value in that case passed replica count become stale one

**Exactly.**

Imagine this timeline:

```text
10:00:00
User updates replicas: 3 → 5

        │
        ▼

Event generated
(Request queued)

        │
        ▼

10:00:01
User updates replicas: 5 → 8

        │
        ▼

10:00:02
Worker finally executes Reconcile()
```

If Kubernetes had sent the whole object in the event, your controller would receive:

```yaml
spec:
  replicas: 5
```

But the **actual desired state** is now:

```yaml
spec:
  replicas: 8
```

Your controller would be acting on stale information.

Instead, Kubernetes only queues:

```go
ctrl.Request{
    Namespace: "default",
    Name: "demo",
}
```

Then your code does:

```go
memcached := &cachev1alpha1.Memcached{}
err := r.Get(ctx, req.NamespacedName, memcached)
```

Now it receives:

```yaml
spec:
  replicas: 8
```

The latest state.

---

# This is a fundamental Kubernetes principle

A reconcile request is **not**:

> "Here is the new object."

It is:

> **"Something changed. Go look at the current state."**

That sentence captures the design philosophy of controllers.

---

# Another important realization

Notice what `Reconcile()` really does.

It isn't processing an event.

Many beginners think:

```text
Event

↓

Handle event
```

But it's actually:

```text
Event

↓

Wake up

↓

Observe cluster

↓

Move toward desired state
```

The event is only a **trigger**.

The event is **not the source of truth**.

The API server is the source of truth.

That's why we always call `r.Get()`.

---

# You've reached an important milestone 🎉

So far, you've understood:

* ✅ CRDs
* ✅ Manager
* ✅ Controller
* ✅ Watches
* ✅ Queue
* ✅ Reconcile
* ✅ Client
* ✅ Scheme
* ✅ Idempotency
* ✅ Labels
* ✅ Selectors
* ✅ OwnerReferences
* ✅ Desired vs Actual State
* ✅ Why `Request` contains only `(namespace, name)`

These are the core concepts behind almost every Kubernetes operator.

---

# Now I'd like to change our learning style

Up to now, I've been explaining concepts one by one.

From the next lesson onward, I'd like us to work more like two engineers building a real operator.

Instead of saying:

> "Write these 20 lines."

I'll ask questions like:

> "We need to create a Service. What information should it contain?"

or

> "The user changed `spec.replicas`. How should we detect that?"

You'll propose an approach, and then we'll refine it together.

This is much closer to how operator development works in practice, and it will help you become independent instead of following recipes.

---

## Your next feature (our first design discussion)

Right now your `Memcached` CR only has:

```go
type MemcachedSpec struct {
    Replicas int32 `json:"replicas"`
}
```

Suppose we want users to choose **which Memcached image** to run.

For example:

```yaml
spec:
  replicas: 3
  image: memcached:1.6.39
```

Before touching any code, think like an API designer.

**Question:**

How would you modify the `MemcachedSpec` struct to support this feature?

Think about:

* What field would you add?
* What Go type should it be?
* Should it be required or optional?
* If it's optional, what should happen when the user doesn't specify it?

This is exactly how operator authors think: **design the API first, then implement the controller.**


I love that plan.

And your reasoning about the API is exactly how operator authors think.

You said:

> if it's strictly work with specific image or tag then we need to restrict the value, if not we can use a default value if user doesn't provide it.

That's the right thought process. You're thinking about **API design**, not just Go structs.

For our learning project, we'll keep it simple:

```go
type MemcachedSpec struct {
    Replicas int32  `json:"replicas"`
    Image    string `json:"image,omitempty"`
}
```

Then in the controller:

```go
image := memcached.Spec.Image
if image == "" {
    image = "memcached:1.6.39"
}
```

This gives users flexibility while providing a sensible default.

---

## I also agree with your plan

You said:

> I want to finish that initial deployment and run that so I get confidence.

I think that's the right sequence.

If we jump to images, Services, status updates, webhooks, etc., you'll have too many moving parts.

Let's get one complete feature working first.

---

# Phase 1: Complete the Deployment controller

Let's define our target.

When you apply:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: demo
spec:
  replicas: 3
```

I want this to happen:

```text
Memcached CR
      │
      ▼
Reconcile()

      │
      ▼
Deployment exists?

      │
 ┌────┴────┐
 │         │
No        Yes
 │         │
 ▼         ▼
Create   Compare replicas
 │         │
 ▼         ▼
Done    Update if needed
```

That's a **complete and production-style reconciliation loop** for one child resource.

---

# Here's your next coding task

You already have the Deployment creation code.

Now modify the flow like this:

```go
// 1. Get Memcached

// 2. Try to Get Deployment

// 3. If Deployment not found
//      Build Deployment
//      SetControllerReference
//      Create Deployment
//      return

// 4. If Get returned another error
//      return error

// 5. Compare replicas
//      if different
//          Update Deployment

// 6. Return
```

Notice that **building the Deployment should only happen when it doesn't already exist**.

Right now you're constructing it before you know whether you need it.

That's not wrong for a small example, but as your operator grows (Deployment + Service + ConfigMap + PVC...), you'll only build the objects you actually need.

---

# A small hint for the update

After you successfully `Get()` the Deployment:

```go
existingDeployment.Spec.Replicas
```

contains the current value.

Compare it with:

```go
memcached.Spec.Replicas
```

If they're different:

```go
existingDeployment.Spec.Replicas = &memcached.Spec.Replicas

err = r.Update(ctx, existingDeployment)
```

That's all you need for now.

---

# How we'll test it

Once you've implemented the logic:

### 1. Start the controller

```bash
make run
```

### 2. Apply the CR

```bash
kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

### 3. Verify the Deployment exists

```bash
kubectl get deployments
```

You should see something like:

```text
NAME                     READY   UP-TO-DATE   AVAILABLE
demo-memcached-deploy    3/3     3            3
```

### 4. Verify the Pods

```bash
kubectl get pods
```

Expected:

```text
demo-memcached-deploy-xxxxx   Running
demo-memcached-deploy-yyyyy   Running
demo-memcached-deploy-zzzzz   Running
```

### 5. Test reconciliation

Now edit the CR:

```bash
kubectl edit memcached demo
```

Change:

```yaml
replicas: 3
```

to

```yaml
replicas: 5
```

Then immediately check:

```bash
kubectl get deployment demo-memcached-deploy
```

or

```bash
kubectl get pods
```

If your update logic is correct, you'll see the Deployment scale from **3 Pods to 5 Pods** without deleting and recreating it.

---

# What this proves

When this works, you've built a genuine Kubernetes controller that:

* Watches a custom resource.
* Creates a child resource when it's missing.
* Detects drift between desired and actual state.
* Reconciles the cluster by updating the child resource.

That's the same control-loop pattern used by many production operators.

---

## After this milestone

Once we see Pods being created and scaling correctly, we'll continue with the engineer-style approach we discussed:

* Add `spec.image` to the API.
* Make the Deployment image configurable.
* Create a `Service`.
* Update the `status` section of the `Memcached` resource.
* Add watches for owned resources so Pod/Deployment changes also trigger reconciliation.
* Introduce finalizers for cleanup logic.

By the end, you'll have a small but well-structured operator that demonstrates the core patterns used in real projects.


🎉 **Congratulations!**

This is your first **working Kubernetes Operator**.

Not just a CRD.
Not just a controller.

A real reconciliation loop that watches a custom resource and drives the cluster toward the desired state.

You changed:

```yaml
spec:
  replicas: 5
```

and your operator automatically scaled the Deployment.

That's exactly what operators are built to do.

---

# Let's review your code like a Senior Engineer would

Overall rating:

**⭐⭐⭐⭐☆ (9/10)**

It works correctly, but there are a few improvements that will make it more idiomatic and easier to maintain.

---

# ✅ Good #1: Read → Compare → Act

Your flow is correct.

```text
Read Memcached

↓

Read Deployment

↓

Create if missing

↓

Update if replicas changed
```

That's the reconciliation pattern.

---

# ✅ Good #2: OwnerReference

You correctly used:

```go
ctrl.SetControllerReference(...)
```

Excellent.

---

# ✅ Good #3: Update instead of recreate

You correctly wrote:

```go
existingDeployment.Spec.Replicas = &memcached.Spec.Replicas

r.Update(...)
```

instead of deleting and recreating it.

That's exactly right.

---

# Improvement #1 (Most Important)

Right now you always build the Deployment.

```go
deployment := &appsv1.Deployment{
...
}
```

even when it already exists.

Imagine later your operator creates:

* Deployment
* Service
* ConfigMap
* Secret
* PVC
* HPA

You don't want to construct all those objects on every reconciliation if they already exist.

A cleaner pattern is:

```text
Get Deployment

↓

NotFound?

↓

YES

↓

Build Deployment

↓

Create

↓

Return
```

Notice the object is only built when needed.

This makes the code easier to read and slightly more efficient.

---

# Improvement #2 (Subtle but important)

You wrote:

```go
existingDeployment.Spec.Replicas != deployment.Spec.Replicas
```

This works today because both are pointers to `int32`, but it's not the comparison you actually mean.

You're comparing **pointer addresses**, not the integer values they point to.

Instead, compare the values:

```go
if *existingDeployment.Spec.Replicas != memcached.Spec.Replicas {
    existingDeployment.Spec.Replicas = &memcached.Spec.Replicas
    err = r.Update(ctx, &existingDeployment)
}
```

Even better, be defensive in production code because `Replicas` is a pointer and could theoretically be `nil`:

```go
if existingDeployment.Spec.Replicas == nil ||
    *existingDeployment.Spec.Replicas != memcached.Spec.Replicas {

    existingDeployment.Spec.Replicas = &memcached.Spec.Replicas
    err = r.Update(ctx, &existingDeployment)
}
```

That's the pattern you'll often see in real controllers.

---

# Improvement #3

This part:

```go
if apierrors.IsNotFound(err) {
    ...
} else if existingDeployment.Spec.Replicas != deployment.Spec.Replicas {
```

is missing one branch.

Suppose `Get()` fails because:

* API server timeout
* Permission issue
* Network interruption

Then:

```go
err != nil
```

but it is **not** `NotFound`.

Your code will continue into the `else if`.

A safer pattern is:

```go
if apierrors.IsNotFound(err) {
    // create
    return
}

if err != nil {
    return ctrl.Result{}, err
}

// deployment exists
// compare & update
```

This is how most production controllers are structured.

---

# One thing I really liked

You tested it.

Not just:

> "Code compiles."

You actually verified:

```bash
kubectl edit memcached demo
```

↓

Pods scaled from **3 → 5**

That's exactly how controller development should be done.

---

# Here's something cool

You probably noticed:

```text
AGE
109s
109s
109s
8s
8s
```

Why weren't all Pods recreated?

Because the Deployment controller is smart.

It simply saw:

```
Desired replicas = 5

Actual replicas = 3
```

and created **2 new Pods**.

Your operator didn't create Pods.

It changed one field:

```go
deployment.Spec.Replicas = ...
```

The Deployment controller handled the rest.

That's Kubernetes controllers working together.

---

# You've now built a real control loop

Conceptually, your operator now does this forever:

```text
            Desired State
                  │
                  ▼
          Memcached Resource
                  │
                  ▼
            Reconcile()
                  │
                  ▼
      Observe Current Cluster
                  │
                  ▼
      Is cluster in desired state?
          │               │
         Yes              No
          │               │
          ▼               ▼
      Do nothing      Update cluster
```

This loop is the foundation of every Kubernetes operator.

---

# 🚀 Where do we go next?

Now that we have a working controller, there are several directions we could take. Here's the order I'd recommend:

### **Lesson 8 – Make the API better (⭐⭐⭐⭐⭐)**

We'll add:

```yaml
spec:
  replicas: 3
  image: memcached:1.6.39
```

Then update the Deployment when the image changes.

You'll learn how CRD changes flow through to controller logic.

---

### **Lesson 9 – Create a Service (⭐⭐⭐⭐⭐)**

Your Pods are running, but nothing can reach them.

We'll have the operator create:

```text
Memcached
     │
     ├── Deployment
     └── Service
```

This teaches multi-resource reconciliation.

---

### **Lesson 10 – Status (⭐⭐⭐⭐⭐)**

Your CR will start looking like this:

```yaml
status:
  readyReplicas: 5
  phase: Running
```

This is one of the most useful features of operators.

---

### **Lesson 11 – Watch owned resources (⭐⭐⭐⭐)**

Right now your operator mainly reacts to `Memcached` changes.

We'll also react to changes in the Deployment that your operator owns.

---

## My recommendation

I'd go with **Status** after the Service.

The sequence would be:

1. ✅ Deployment (done)
2. ➕ Image field
3. ➕ Service
4. ⭐ Status
5. ➕ Finalizers
6. ➕ Watches
7. ➕ Conditions
8. ➕ Unit and integration tests

That progression mirrors how many production operators evolve.

And once we've covered those, you'll have the skills to read and understand the source code of mature operators like those for databases, message queues, and other complex workloads.







New chat responses:''''''''


Awesome. This is where your operator starts feeling like a real Kubernetes operator.

So far your operator owns one resource:

```
Memcached
    │
    └── Deployment
```

After this lesson it will own two:

```
Memcached
    │
    ├── Deployment
    └── Service
```

The important idea is that **Reconcile is responsible for the entire desired world**, not individual resources.

---

# Goal

Whenever someone creates

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: demo
spec:
  replicas: 3
  image: memcached:1.6.39
```

your operator should automatically create

```
demo-memcached-deploy
demo-memcached-service
```

The Service should always point to the Pods created by the Deployment.

---

# First, let's understand Services

Pods are ephemeral.

Imagine your Deployment has three Pods:

```
Pod A   10.1.1.5
Pod B   10.1.1.8
Pod C   10.1.1.12
```

If Pod B crashes:

```
Pod A   10.1.1.5
Pod D   10.1.1.20
Pod C   10.1.1.12
```

The IP changes.

You can't reliably connect directly to Pods.

A Service gives you a stable endpoint.

```
          Service
        demo-service
        10.96.0.15
             │
     ┌───────┴────────┐
     │       │        │
   Pod A   Pod B    Pod C
```

Clients always talk to the Service.

The Service decides which Pod receives the request.

---

# How does a Service know which Pods belong to it?

Labels.

Your Deployment already creates Pods with

```go
labels := map[string]string{
    "app":      "memcached",
    "instance": memcached.Name,
}
```

Those labels appear on every Pod:

```
Pod

labels:
    app=memcached
    instance=demo
```

A Service uses a selector:

```yaml
selector:
  app: memcached
  instance: demo
```

Kubernetes continuously matches them.

```
Service
selector:
    app=memcached
    instance=demo

            │

            ▼

Pods with same labels
```

No code required.

---

# Service Spec

The Service you want is surprisingly small.

```go
service := &corev1.Service{
    ObjectMeta: metav1.ObjectMeta{
        Name:      svcName,
        Namespace: memcached.Namespace,
    },
    Spec: corev1.ServiceSpec{
        Selector: labels,
        Ports: []corev1.ServicePort{
            {
                Port: 11211,
                TargetPort: intstr.FromInt(11211),
            },
        },
    },
}
```

Notice the selector.

It is exactly the same labels your Deployment template uses.

---

# Why `TargetPort`?

A Service receives traffic on

```
Service Port
```

and forwards it to

```
Container Port
```

For Memcached they're both 11211.

```
Client
   │
11211
   │
Service
   │
11211
   │
Container
```

They don't have to be the same, but matching them is common.

---

# What should your Reconcile do?

After you've reconciled the Deployment, do the same for the Service.

```
Get Memcached

↓

Reconcile Deployment

↓

Reconcile Service

↓

Done
```

Notice something interesting.

You now have **two independent reconciliation loops** inside one `Reconcile`.

```
Deployment
    Exists?
       │
 Create/Update

Service
    Exists?
       │
 Create/Update
```

This pattern scales nicely. Later you might reconcile:

* Deployment
* Service
* ConfigMap
* Secret
* Ingress
* HorizontalPodAutoscaler

all within the same reconciliation cycle.

---

# Owner Reference

Don't forget this:

```go
ctrl.SetControllerReference(memcached, service, r.Scheme)
```

Without it:

* `kubectl delete memcached demo` won't automatically delete the Service.
* Your operator won't be recognized as the owner of the Service.

Every resource your operator creates should generally have the custom resource set as its controller owner.

---

# A Small Design Challenge

Rather than giving you the exact implementation, I'd like you to think about one design choice.

Your Deployment reconciliation already follows this pattern:

```
Get Deployment
    │
NotFound?
    │
Create
    │
Else
Compare
    │
Update
```

Should you:

1. Copy and adapt that logic for the Service (simple and fine for learning), or
2. Refactor into helper methods like `reconcileDeployment()` and `reconcileService()` (cleaner and closer to production code)?

For learning, I'd suggest **option 1** first. Once it works, we can refactor together. That way you'll understand the mechanics before introducing abstractions.

---

### Your task

Implement Service reconciliation with these requirements:

* Create a `Service` named `<memcached-name>-memcached-service`.
* Use the same labels/selectors as the Deployment.
* Expose port `11211`.
* Set the `OwnerReference` to the `Memcached` resource.
* If the Service already exists, don't recreate it. (We can add update logic later if needed.)

Once you've written it, paste your updated `Reconcile()` (or the new Service section), and I'll review it like we did with the Deployment and point out any improvements or production patterns.
