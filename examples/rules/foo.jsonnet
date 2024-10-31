{
  groups: [
    {
      name: 'foo',
      limit: 1,
      rules: [
        {
          alert: 'foo',
          expr: |||
            # ignore_validations: hasLabels,hasAnyOfAnnotations,hasAnnotations,hasAllowedLimit
            1
          |||,
        },
      ],
    },
  ],
}
