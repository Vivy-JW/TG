using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000042 RID: 66
	public class ContainerMapManager
	{
		// Token: 0x17000084 RID: 132
		// (get) Token: 0x0600021F RID: 543 RVA: 0x0000961A File Offset: 0x0000781A
		public static ContainerMapManager This
		{
			get
			{
				if (ContainerMapManager._this == null)
				{
					ContainerMapManager._this = new ContainerMapManager();
				}
				return ContainerMapManager._this;
			}
		}

		// Token: 0x06000221 RID: 545 RVA: 0x0000963C File Offset: 0x0000783C
		private void SetMapArea(int w, int h)
		{
			this._w = w;
			this._h = h;
			this.map = new ContainerMapManager.Cell[this._w, this._h];
			for (int i = 0; i < w; i++)
			{
				for (int j = 0; j < h; j++)
				{
					this.map[i, j] = new ContainerMapManager.Cell();
				}
			}
		}

		// Token: 0x06000222 RID: 546 RVA: 0x00009698 File Offset: 0x00007898
		public void CheckOverlapping(XmlElement container)
		{
			this.SetMapArea(container.GridWidth, container.GridHeight);
			foreach (XmlElement xmlElement in container.Nodes)
			{
				xmlElement.IsPosFine = true;
				int num = 0;
				while (num < xmlElement.GridHeight && xmlElement.GridY + num < this._h && xmlElement.GridX < this._w)
				{
					if (xmlElement.GridX > 0 && this.map[xmlElement.GridX - 1, xmlElement.GridY + num].IsTagged())
					{
						this.map[xmlElement.GridX - 1, xmlElement.GridY + num].SetBeNext(container, xmlElement);
					}
					if (xmlElement.GridX + xmlElement.GridWidth < this._w && this.map[xmlElement.GridX + xmlElement.GridWidth, xmlElement.GridY].IsTagged())
					{
						this.map[xmlElement.GridX + xmlElement.GridWidth, xmlElement.GridY].SetBeNext(container, xmlElement);
					}
					int num2 = 0;
					while (num2 < xmlElement.GridWidth && xmlElement.GridX + num2 < this._w)
					{
						this.map[xmlElement.GridX + num2, xmlElement.GridY + num].SetOwner(xmlElement);
						num2++;
					}
					num++;
				}
			}
			this.Clear();
		}

		// Token: 0x06000223 RID: 547 RVA: 0x00009838 File Offset: 0x00007A38
		private void Clear()
		{
			Array.Clear(this.map, 0, this.map.Length);
		}

		// Token: 0x040000C7 RID: 199
		private static ContainerMapManager _this;

		// Token: 0x040000C8 RID: 200
		private int _w;

		// Token: 0x040000C9 RID: 201
		private int _h;

		// Token: 0x040000CA RID: 202
		private ContainerMapManager.Cell[,] map;

		// Token: 0x02000043 RID: 67
		private class Cell
		{
			// Token: 0x17000085 RID: 133
			// (get) Token: 0x06000225 RID: 549 RVA: 0x00009853 File Offset: 0x00007A53
			// (set) Token: 0x06000226 RID: 550 RVA: 0x0000985B File Offset: 0x00007A5B
			private byte TAG { get; set; }

			// Token: 0x06000227 RID: 551 RVA: 0x00009864 File Offset: 0x00007A64
			public Cell()
			{
				this.TAG = ContainerMapManager.Cell.UNTAGGED;
			}

			// Token: 0x06000228 RID: 552 RVA: 0x00009877 File Offset: 0x00007A77
			public bool IsTagged()
			{
				return this.TAG == ContainerMapManager.Cell.TAGGED;
			}

			// Token: 0x06000229 RID: 553 RVA: 0x00009888 File Offset: 0x00007A88
			public void SetOwner(XmlElement owner)
			{
				if (this.Owners == null)
				{
					this.Owners = new List<XmlElement>();
				}
				if (this.Owners.Contains(owner))
				{
					return;
				}
				this.Owners.Add(owner);
				this.TAG = ContainerMapManager.Cell.TAGGED;
				if (this.Owners.Count > 1)
				{
					foreach (XmlElement xmlElement in this.Owners)
					{
						xmlElement.IsPosFine = false;
					}
				}
			}

			// Token: 0x0600022A RID: 554 RVA: 0x00009924 File Offset: 0x00007B24
			public void SetBeNext(XmlElement container, XmlElement element)
			{
				foreach (XmlElement xmlElement in this.Owners)
				{
					ComponentType type = container.Type;
					if (type == ComponentType.Grid)
					{
						switch (xmlElement.Type)
						{
						case ComponentType.Group:
						case ComponentType.ScrollGrid:
						case ComponentType.Table:
						case ComponentType.Tree:
							continue;
						}
					}
					xmlElement.IsPosFine = false;
					element.IsPosFine = false;
				}
			}

			// Token: 0x040000CB RID: 203
			private static byte UNTAGGED = 1;

			// Token: 0x040000CC RID: 204
			private static byte TAGGED = 0;

			// Token: 0x040000CD RID: 205
			private List<XmlElement> Owners;
		}
	}
}
