using System;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000033 RID: 51
	public interface IParameterModel
	{
		// Token: 0x1700006E RID: 110
		// (get) Token: 0x06000150 RID: 336
		string Name { get; }

		// Token: 0x1700006F RID: 111
		// (get) Token: 0x06000151 RID: 337
		string Description { get; }

		// Token: 0x17000070 RID: 112
		// (get) Token: 0x06000152 RID: 338
		string Type { get; }

		// Token: 0x17000071 RID: 113
		// (get) Token: 0x06000153 RID: 339
		UsageType Usage { get; }
	}
}
