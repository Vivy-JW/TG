using System;
using System.Collections.Generic;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200002B RID: 43
	public interface IMethod
	{
		// Token: 0x17000053 RID: 83
		// (get) Token: 0x0600010A RID: 266
		IEnumerable<IParameterModel> Parameters { get; }

		// Token: 0x17000054 RID: 84
		// (get) Token: 0x0600010B RID: 267
		// (set) Token: 0x0600010C RID: 268
		string Description { get; set; }

		// Token: 0x17000055 RID: 85
		// (get) Token: 0x0600010D RID: 269
		// (set) Token: 0x0600010E RID: 270
		string Name { get; set; }

		// Token: 0x0600010F RID: 271
		IEnumerable<string> GetParameterDescription();
	}
}
