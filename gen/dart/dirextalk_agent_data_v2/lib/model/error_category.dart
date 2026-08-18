//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;


class ErrorCategory {
  /// Instantiate a new enum with the provided [value].
  const ErrorCategory._(this.value);

  /// The underlying value of this enum member.
  final String value;

  @override
  String toString() => value;

  String toJson() => value;

  static const authentication = ErrorCategory._(r'authentication');
  static const authorization = ErrorCategory._(r'authorization');
  static const validation = ErrorCategory._(r'validation');
  static const notFound = ErrorCategory._(r'not_found');
  static const conflict = ErrorCategory._(r'conflict');
  static const rateLimit = ErrorCategory._(r'rate_limit');
  static const unavailable = ErrorCategory._(r'unavailable');
  static const upstream = ErrorCategory._(r'upstream');
  static const internal = ErrorCategory._(r'internal');

  /// List of all possible values in this [enum][ErrorCategory].
  static const values = <ErrorCategory>[
    authentication,
    authorization,
    validation,
    notFound,
    conflict,
    rateLimit,
    unavailable,
    upstream,
    internal,
  ];

  static ErrorCategory? fromJson(dynamic value) => ErrorCategoryTypeTransformer().decode(value);

  static List<ErrorCategory> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <ErrorCategory>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ErrorCategory.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ErrorCategory] to String,
/// and [decode] dynamic data back to [ErrorCategory].
class ErrorCategoryTypeTransformer {
  factory ErrorCategoryTypeTransformer() => _instance ??= const ErrorCategoryTypeTransformer._();

  const ErrorCategoryTypeTransformer._();

  String encode(ErrorCategory data) => data.value;

  /// Decodes a [dynamic value][data] to a ErrorCategory.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ErrorCategory? decode(dynamic data, {bool allowNull = true}) {
    if (data != null) {
      switch (data) {
        case r'authentication': return ErrorCategory.authentication;
        case r'authorization': return ErrorCategory.authorization;
        case r'validation': return ErrorCategory.validation;
        case r'not_found': return ErrorCategory.notFound;
        case r'conflict': return ErrorCategory.conflict;
        case r'rate_limit': return ErrorCategory.rateLimit;
        case r'unavailable': return ErrorCategory.unavailable;
        case r'upstream': return ErrorCategory.upstream;
        case r'internal': return ErrorCategory.internal;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// Singleton [ErrorCategoryTypeTransformer] instance.
  static ErrorCategoryTypeTransformer? _instance;
}

