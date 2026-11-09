using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x0200004B RID: 75
	public class ChangeSizeUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000284 RID: 644 RVA: 0x0000B4F0 File Offset: 0x000096F0
		public ChangeSizeUndoRedoCommand(IEnumerable<XmlElement> elements)
		{
			this._elements = new Dictionary<XmlElement, ChangeSizeUndoRedoCommand.ComponentDimension>();
			foreach (XmlElement xmlElement in elements)
			{
				if (null == this._key)
				{
					this._key = xmlElement.Key;
				}
				ChangeSizeUndoRedoCommand.ComponentDimension componentDimension = new ChangeSizeUndoRedoCommand.ComponentDimension(xmlElement.GridX, xmlElement.GridY, xmlElement.GridWidth, xmlElement.GridHeight);
				this._elements.Add(xmlElement, componentDimension);
			}
		}

		// Token: 0x06000285 RID: 645 RVA: 0x0000B588 File Offset: 0x00009788
		public void SetFinalSize(XmlElement element, int x, int y, int width, int height)
		{
			foreach (KeyValuePair<XmlElement, ChangeSizeUndoRedoCommand.ComponentDimension> keyValuePair in this._elements)
			{
				if (element == keyValuePair.Key)
				{
					keyValuePair.Value.SetFinalDimension(x, y, width, height);
				}
			}
		}

		// Token: 0x06000286 RID: 646 RVA: 0x0000B5F0 File Offset: 0x000097F0
		public override void Undo()
		{
			foreach (KeyValuePair<XmlElement, ChangeSizeUndoRedoCommand.ComponentDimension> keyValuePair in this._elements)
			{
				XmlElement key = keyValuePair.Key;
				if (keyValuePair.Value.oriX > key.GridX || keyValuePair.Value.oriY > key.GridY)
				{
					key.GridWidth = keyValuePair.Value.oriWidth;
					key.GridHeight = keyValuePair.Value.oriHeight;
					key.GridX = keyValuePair.Value.oriX;
					key.GridY = keyValuePair.Value.oriY;
				}
				else
				{
					key.GridX = keyValuePair.Value.oriX;
					key.GridY = keyValuePair.Value.oriY;
					key.GridWidth = keyValuePair.Value.oriWidth;
					key.GridHeight = keyValuePair.Value.oriHeight;
				}
			}
			ComponentHelper.Get(this._key).MultipleSelection(this._elements.Keys);
		}

		// Token: 0x06000287 RID: 647 RVA: 0x0000B720 File Offset: 0x00009920
		public override void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this, false);
			SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (KeyValuePair<XmlElement, ChangeSizeUndoRedoCommand.ComponentDimension> keyValuePair in this._elements)
			{
				XmlElement key = keyValuePair.Key;
				if (keyValuePair.Value.finX > key.GridX || keyValuePair.Value.finY > key.GridY)
				{
					key.GridWidth = keyValuePair.Value.finWidth;
					key.GridHeight = keyValuePair.Value.finHeight;
					key.GridX = keyValuePair.Value.finX;
					key.GridY = keyValuePair.Value.finY;
				}
				else
				{
					key.GridX = keyValuePair.Value.finX;
					key.GridY = keyValuePair.Value.finY;
					key.GridWidth = keyValuePair.Value.finWidth;
					key.GridHeight = keyValuePair.Value.finHeight;
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._key).MultipleSelection(this._elements.Keys);
		}

		// Token: 0x06000288 RID: 648 RVA: 0x0000B884 File Offset: 0x00009A84
		public override void Clear()
		{
			this._elements.Clear();
		}

		// Token: 0x040000F5 RID: 245
		private PackageKey _key;

		// Token: 0x040000F6 RID: 246
		private Dictionary<XmlElement, ChangeSizeUndoRedoCommand.ComponentDimension> _elements;

		// Token: 0x0200004C RID: 76
		private class ComponentDimension
		{
			// Token: 0x1700009D RID: 157
			// (get) Token: 0x06000289 RID: 649 RVA: 0x0000B891 File Offset: 0x00009A91
			// (set) Token: 0x0600028A RID: 650 RVA: 0x0000B899 File Offset: 0x00009A99
			public int oriX { get; private set; }

			// Token: 0x1700009E RID: 158
			// (get) Token: 0x0600028B RID: 651 RVA: 0x0000B8A2 File Offset: 0x00009AA2
			// (set) Token: 0x0600028C RID: 652 RVA: 0x0000B8AA File Offset: 0x00009AAA
			public int oriY { get; private set; }

			// Token: 0x1700009F RID: 159
			// (get) Token: 0x0600028D RID: 653 RVA: 0x0000B8B3 File Offset: 0x00009AB3
			// (set) Token: 0x0600028E RID: 654 RVA: 0x0000B8BB File Offset: 0x00009ABB
			public int oriWidth { get; private set; }

			// Token: 0x170000A0 RID: 160
			// (get) Token: 0x0600028F RID: 655 RVA: 0x0000B8C4 File Offset: 0x00009AC4
			// (set) Token: 0x06000290 RID: 656 RVA: 0x0000B8CC File Offset: 0x00009ACC
			public int oriHeight { get; private set; }

			// Token: 0x170000A1 RID: 161
			// (get) Token: 0x06000291 RID: 657 RVA: 0x0000B8D5 File Offset: 0x00009AD5
			// (set) Token: 0x06000292 RID: 658 RVA: 0x0000B8DD File Offset: 0x00009ADD
			public int finX { get; private set; }

			// Token: 0x170000A2 RID: 162
			// (get) Token: 0x06000293 RID: 659 RVA: 0x0000B8E6 File Offset: 0x00009AE6
			// (set) Token: 0x06000294 RID: 660 RVA: 0x0000B8EE File Offset: 0x00009AEE
			public int finY { get; private set; }

			// Token: 0x170000A3 RID: 163
			// (get) Token: 0x06000295 RID: 661 RVA: 0x0000B8F7 File Offset: 0x00009AF7
			// (set) Token: 0x06000296 RID: 662 RVA: 0x0000B8FF File Offset: 0x00009AFF
			public int finWidth { get; private set; }

			// Token: 0x170000A4 RID: 164
			// (get) Token: 0x06000297 RID: 663 RVA: 0x0000B908 File Offset: 0x00009B08
			// (set) Token: 0x06000298 RID: 664 RVA: 0x0000B910 File Offset: 0x00009B10
			public int finHeight { get; private set; }

			// Token: 0x06000299 RID: 665 RVA: 0x0000B919 File Offset: 0x00009B19
			public ComponentDimension(int x, int y, int width, int height)
			{
				this.oriX = x;
				this.oriY = y;
				this.oriWidth = width;
				this.oriHeight = height;
			}

			// Token: 0x0600029A RID: 666 RVA: 0x0000B93E File Offset: 0x00009B3E
			public void SetFinalDimension(int x, int y, int width, int height)
			{
				this.finX = x;
				this.finY = y;
				this.finWidth = width;
				this.finHeight = height;
			}
		}
	}
}
