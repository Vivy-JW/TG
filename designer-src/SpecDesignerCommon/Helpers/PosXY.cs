using System;
using System.Windows;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000107 RID: 263
	public struct PosXY
	{
		// Token: 0x0600092B RID: 2347 RVA: 0x0002C2F1 File Offset: 0x0002A4F1
		public PosXY(Point p)
		{
			this.PosX = (int)p.X;
			this.PosY = (int)p.Y;
		}

		// Token: 0x0600092C RID: 2348 RVA: 0x0002C30F File Offset: 0x0002A50F
		public PosXY(int x, int y)
		{
			this.PosX = x;
			this.PosY = y;
		}

		// Token: 0x04000345 RID: 837
		public int PosX;

		// Token: 0x04000346 RID: 838
		public int PosY;
	}
}
