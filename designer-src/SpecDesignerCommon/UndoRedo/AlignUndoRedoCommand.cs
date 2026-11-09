using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000019 RID: 25
	public class AlignUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x060000B8 RID: 184 RVA: 0x00004F40 File Offset: 0x00003140
		public AlignUndoRedoCommand(List<XmlElement> elements, AlignOptions option)
		{
			this._option = option;
			this._elements = new Dictionary<XmlElement, AlignUndoRedoCommand.ComponentDimension>();
			foreach (XmlElement xmlElement in elements)
			{
				if (null == this._key)
				{
					this._key = xmlElement.Key;
				}
				AlignUndoRedoCommand.ComponentDimension componentDimension = new AlignUndoRedoCommand.ComponentDimension(xmlElement.GridX, xmlElement.GridY, xmlElement.GridWidth, xmlElement.GridHeight);
				this._elements.Add(xmlElement, componentDimension);
			}
		}

		// Token: 0x060000B9 RID: 185 RVA: 0x00004FE4 File Offset: 0x000031E4
		public void Undo()
		{
			foreach (KeyValuePair<XmlElement, AlignUndoRedoCommand.ComponentDimension> keyValuePair in this._elements)
			{
				XmlElement key = keyValuePair.Key;
				switch (this._option)
				{
				case AlignOptions.STRETCH:
					key.GridWidth = keyValuePair.Value.Width;
					key.GridHeight = keyValuePair.Value.Height;
					key.GridX = keyValuePair.Value.X;
					key.GridY = keyValuePair.Value.Y;
					break;
				case AlignOptions.LEFT:
				case AlignOptions.RIGHT:
					key.GridX = keyValuePair.Value.X;
					break;
				case AlignOptions.TOP:
				case AlignOptions.BOTTOM:
					key.GridY = keyValuePair.Value.Y;
					break;
				}
			}
			ComponentHelper.Get(this._key).MultipleSelection(this._elements.Keys);
		}

		// Token: 0x060000BA RID: 186 RVA: 0x000050F0 File Offset: 0x000032F0
		public void Execute()
		{
			switch (this._option)
			{
			case AlignOptions.STRETCH:
			{
				int num = int.MaxValue;
				int num2 = 0;
				foreach (XmlElement xmlElement in this._elements.Keys)
				{
					num = Math.Min(num, xmlElement.GridX);
					num2 = Math.Max(num2, xmlElement.GridX + xmlElement.GridWidth);
				}
				int num3 = num2 - num;
				using (Dictionary<XmlElement, AlignUndoRedoCommand.ComponentDimension>.KeyCollection.Enumerator enumerator2 = this._elements.Keys.GetEnumerator())
				{
					while (enumerator2.MoveNext())
					{
						XmlElement xmlElement2 = enumerator2.Current;
						xmlElement2.GridX = num;
						xmlElement2.GridWidth = num3;
					}
					goto IL_0326;
				}
				break;
			}
			case AlignOptions.LEFT:
				break;
			case AlignOptions.RIGHT:
				goto IL_0164;
			case AlignOptions.TOP:
				goto IL_01FF;
			case AlignOptions.BOTTOM:
				goto IL_028E;
			default:
				goto IL_0326;
			}
			int num4 = int.MaxValue;
			foreach (XmlElement xmlElement3 in this._elements.Keys)
			{
				num4 = Math.Min(num4, xmlElement3.GridX);
			}
			using (Dictionary<XmlElement, AlignUndoRedoCommand.ComponentDimension>.KeyCollection.Enumerator enumerator4 = this._elements.Keys.GetEnumerator())
			{
				while (enumerator4.MoveNext())
				{
					XmlElement xmlElement4 = enumerator4.Current;
					xmlElement4.GridX = num4;
				}
				goto IL_0326;
			}
			IL_0164:
			int num5 = 0;
			foreach (XmlElement xmlElement5 in this._elements.Keys)
			{
				num5 = Math.Max(num5, xmlElement5.GridX + xmlElement5.GridWidth);
			}
			using (Dictionary<XmlElement, AlignUndoRedoCommand.ComponentDimension>.KeyCollection.Enumerator enumerator6 = this._elements.Keys.GetEnumerator())
			{
				while (enumerator6.MoveNext())
				{
					XmlElement xmlElement6 = enumerator6.Current;
					xmlElement6.GridX = num5 - xmlElement6.GridWidth;
				}
				goto IL_0326;
			}
			IL_01FF:
			int num6 = int.MaxValue;
			foreach (XmlElement xmlElement7 in this._elements.Keys)
			{
				num6 = Math.Min(num6, xmlElement7.GridY);
			}
			using (Dictionary<XmlElement, AlignUndoRedoCommand.ComponentDimension>.KeyCollection.Enumerator enumerator8 = this._elements.Keys.GetEnumerator())
			{
				while (enumerator8.MoveNext())
				{
					XmlElement xmlElement8 = enumerator8.Current;
					xmlElement8.GridY = num6;
				}
				goto IL_0326;
			}
			IL_028E:
			int num7 = 0;
			foreach (XmlElement xmlElement9 in this._elements.Keys)
			{
				num7 = Math.Max(num7, xmlElement9.GridY + xmlElement9.GridHeight);
			}
			foreach (XmlElement xmlElement10 in this._elements.Keys)
			{
				xmlElement10.GridY = num7 - xmlElement10.GridHeight;
			}
			IL_0326:
			ComponentHelper.Get(this._key).MultipleSelection(this._elements.Keys);
		}

		// Token: 0x060000BB RID: 187 RVA: 0x000054BC File Offset: 0x000036BC
		public void Clear()
		{
			this._elements.Clear();
			this._option = AlignOptions.NONE;
		}

		// Token: 0x04000042 RID: 66
		private Dictionary<XmlElement, AlignUndoRedoCommand.ComponentDimension> _elements;

		// Token: 0x04000043 RID: 67
		private AlignOptions _option;

		// Token: 0x04000044 RID: 68
		private PackageKey _key;

		// Token: 0x0200001A RID: 26
		private class ComponentDimension
		{
			// Token: 0x17000022 RID: 34
			// (get) Token: 0x060000BC RID: 188 RVA: 0x000054D0 File Offset: 0x000036D0
			// (set) Token: 0x060000BD RID: 189 RVA: 0x000054D8 File Offset: 0x000036D8
			public int X { get; private set; }

			// Token: 0x17000023 RID: 35
			// (get) Token: 0x060000BE RID: 190 RVA: 0x000054E1 File Offset: 0x000036E1
			// (set) Token: 0x060000BF RID: 191 RVA: 0x000054E9 File Offset: 0x000036E9
			public int Y { get; private set; }

			// Token: 0x17000024 RID: 36
			// (get) Token: 0x060000C0 RID: 192 RVA: 0x000054F2 File Offset: 0x000036F2
			// (set) Token: 0x060000C1 RID: 193 RVA: 0x000054FA File Offset: 0x000036FA
			public int Width { get; private set; }

			// Token: 0x17000025 RID: 37
			// (get) Token: 0x060000C2 RID: 194 RVA: 0x00005503 File Offset: 0x00003703
			// (set) Token: 0x060000C3 RID: 195 RVA: 0x0000550B File Offset: 0x0000370B
			public int Height { get; private set; }

			// Token: 0x060000C4 RID: 196 RVA: 0x00005514 File Offset: 0x00003714
			public ComponentDimension(int x, int y, int width, int height)
			{
				this.X = x;
				this.Y = y;
				this.Width = width;
				this.Height = height;
			}
		}
	}
}
