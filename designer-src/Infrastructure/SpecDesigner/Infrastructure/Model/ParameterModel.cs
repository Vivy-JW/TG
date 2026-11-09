using System;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200004C RID: 76
	public class ParameterModel : IParameterModel
	{
		// Token: 0x170000C9 RID: 201
		// (get) Token: 0x06000248 RID: 584 RVA: 0x0000B1E3 File Offset: 0x000093E3
		// (set) Token: 0x06000249 RID: 585 RVA: 0x0000B1EB File Offset: 0x000093EB
		public string Name { get; set; }

		// Token: 0x170000CA RID: 202
		// (get) Token: 0x0600024A RID: 586 RVA: 0x0000B1F4 File Offset: 0x000093F4
		// (set) Token: 0x0600024B RID: 587 RVA: 0x0000B1FC File Offset: 0x000093FC
		public string Description { get; set; }

		// Token: 0x170000CB RID: 203
		// (get) Token: 0x0600024C RID: 588 RVA: 0x0000B205 File Offset: 0x00009405
		// (set) Token: 0x0600024D RID: 589 RVA: 0x0000B20D File Offset: 0x0000940D
		public string Type { get; set; }

		// Token: 0x170000CC RID: 204
		// (get) Token: 0x0600024E RID: 590 RVA: 0x0000B216 File Offset: 0x00009416
		// (set) Token: 0x0600024F RID: 591 RVA: 0x0000B21E File Offset: 0x0000941E
		public UsageType Usage { get; set; }

		// Token: 0x06000250 RID: 592 RVA: 0x0000B227 File Offset: 0x00009427
		public ParameterModel()
		{
		}

		// Token: 0x06000251 RID: 593 RVA: 0x0000B22F File Offset: 0x0000942F
		public ParameterModel(string name, string type, UsageType ut)
			: this()
		{
			this.Name = name;
			this.Type = type;
			this.Usage = ut;
		}
	}
}
