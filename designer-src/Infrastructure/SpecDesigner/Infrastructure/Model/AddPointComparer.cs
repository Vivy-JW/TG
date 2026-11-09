using System;
using System.Collections.Generic;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000027 RID: 39
	public class AddPointComparer : IEqualityComparer<AddPointModel>
	{
		// Token: 0x060000D5 RID: 213 RVA: 0x00004C58 File Offset: 0x00002E58
		public bool Equals(AddPointModel x, AddPointModel y)
		{
			return x.ToString() == y.ToString() && x.CiteSetting == y.CiteSetting && x.IsLoaded == y.IsLoaded && x.ProgramKey == y.ProgramKey && x.Status == y.Status;
		}

		// Token: 0x060000D6 RID: 214 RVA: 0x00004CB8 File Offset: 0x00002EB8
		public int GetHashCode(AddPointModel obj)
		{
			return obj.GetHashCode();
		}
	}
}
