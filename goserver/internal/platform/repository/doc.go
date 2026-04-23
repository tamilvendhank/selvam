// Package repository defines storage contracts for the newer domain model.
//
// These interfaces intentionally avoid Mongo query primitives so that implementations can
// evolve independently from service/domain code, while still fitting the existing ObjectID-based model.
package repository
